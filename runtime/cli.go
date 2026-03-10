package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/robomotionio/robomotion-go/message"
)

// CLICommandInfo describes a single CLI command derived from a tool-enabled node.
type CLICommandInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	NodeID      string            `json:"node_id"`
	Parameters  []CLIParamInfo    `json:"parameters"`
	Outputs     []CLIOutputInfo   `json:"outputs"`
}

// CLIParamInfo describes one input parameter for a CLI command.
type CLIParamInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// CLIOutputInfo describes one output field for a CLI command.
type CLIOutputInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// RunCLI handles CLI invocation: parses command name and flags, dispatches to the
// matching node, runs its lifecycle, and prints JSON output to stdout.
func RunCLI(args []string) {
	if len(args) == 0 {
		cliError("no command specified; use --list-commands to see available commands")
		return
	}

	commandName := args[0]

	// Special commands
	switch commandName {
	case "--list-commands":
		listCommands()
		return
	case "--skill-md":
		config := ReadConfigFile()
		name := config.Get("name").String()
		version := config.Get("version").String()
		generateSkillMD(name, version, config)
		return
	case "--help", "-h":
		printCLIUsage()
		return
	}

	// Register all node factories (needed for reflection)
	RegisterFactories()

	// Build tool name → node type mapping
	commands := buildCommandMap()

	cmd, ok := commands[commandName]
	if !ok {
		available := make([]string, 0, len(commands))
		for name := range commands {
			available = append(available, name)
		}
		cliError("unknown command %q; available commands: %s", commandName, strings.Join(available, ", "))
		return
	}

	// Parse --key=value flags from remaining args
	flags, err := parseFlags(args[1:])
	if err != nil {
		cliError("%v", err)
		return
	}

	// Extract global flags
	vaultID := flags["vault-id"]
	itemID := flags["item-id"]
	delete(flags, "vault-id")
	delete(flags, "item-id")

	// Extract session flags
	sessionStart := flags["session"] == "true"
	sessionID := flags["session-id"]
	sessionClose := flags["session-close"]
	timeoutStr := flags["session-timeout"]
	delete(flags, "session")
	delete(flags, "session-id")
	delete(flags, "session-close")
	delete(flags, "session-timeout")

	// Session close: tell daemon to shut down
	if sessionClose != "" {
		CloseSession(sessionClose)
		return
	}

	// Session reuse: send command to existing daemon
	if sessionID != "" {
		// Pass vault flags through for credential config injection
		if vaultID != "" {
			flags["vault-id"] = vaultID
		}
		if itemID != "" {
			flags["item-id"] = itemID
		}
		RunSessionClient(sessionID, commandName, flags)
		return
	}

	// Session start: fork daemon, then send first command as client
	if sessionStart {
		timeout := parseSessionTimeout(timeoutStr)
		id := generateSessionID()
		StartDaemonProcess(id, timeout, vaultID, itemID, os.Args)

		// Send first command to the new daemon
		if vaultID != "" {
			flags["vault-id"] = vaultID
		}
		if itemID != "" {
			flags["item-id"] = itemID
		}
		RunSessionClient(id, commandName, flags)
		return
	}

	// Set up CLI runtime helper (replaces gRPC client)
	cliHelper := NewCLIRuntimeHelper()

	// Handle credentials from vault flags — fetch from vault API
	if vaultID != "" && itemID != "" {
		vaultClient, err := NewCLIVaultClient()
		if err != nil {
			cliError("vault auth error: %v", err)
			return
		}
		creds, err := vaultClient.FetchVaultItem(vaultID, itemID)
		if err != nil {
			cliError("vault fetch error: %v", err)
			return
		}
		cliHelper.SetCredentials(creds)
	}

	// Set the global client so all existing code (InVariable.Get, Credential.Get, etc.) works
	client = cliHelper

	// Instantiate the node
	nodeVal := reflect.New(cmd.nodeType)
	handler := nodeVal.Interface().(MessageHandler)

	// Build synthetic config JSON to initialize the node
	nodeConfig := map[string]interface{}{
		"guid": "cli-node",
		"name": commandName,
	}

	// If vault flags provided, inject credential config for Credential fields
	if vaultID != "" && itemID != "" {
		injectCredentialConfig(cmd.nodeType, nodeConfig, vaultID, itemID)
	}

	configJSON, _ := json.Marshal(nodeConfig)
	if err := json.Unmarshal(configJSON, handler); err != nil {
		cliError("failed to initialize node: %v", err)
		return
	}

	// Build message context from flags
	msgData := buildMessageContext(cmd.nodeType, flags)
	msgJSON, _ := json.Marshal(msgData)
	ctx := message.NewContext(msgJSON)

	// Run node lifecycle: OnCreate → OnMessage → OnClose
	if err := handler.OnCreate(); err != nil {
		cliError("OnCreate failed: %v", err)
		return
	}

	err = handler.OnMessage(ctx)

	closeErr := handler.OnClose()

	if err != nil {
		cliError("%v", err)
		return
	}
	if closeErr != nil {
		cliError("OnClose failed: %v", closeErr)
		return
	}

	// Collect output variables from the context
	output := collectCLIOutput(cmd.nodeType, handler, ctx)

	// Print JSON result to stdout
	result, _ := json.Marshal(output)
	fmt.Println(string(result))
}

// commandEntry maps a tool name to its node type and metadata.
type commandEntry struct {
	nodeType    reflect.Type
	nodeID      string
	toolName    string
	description string
}

// buildCommandMap scans all registered node types and builds a map of
// tool_name → commandEntry for nodes that have an embedded Tool field.
func buildCommandMap() map[string]commandEntry {
	commands := make(map[string]commandEntry)
	types := GetNodeTypes()

	for _, t := range types {
		toolField, hasToolField := t.FieldByName("Tool")
		if !hasToolField {
			continue
		}

		toolTag := toolField.Tag.Get("tool")
		toolParts := parseSpec(toolTag)
		toolName := toolParts["name"]
		if toolName == "" {
			continue
		}

		nodeField, _ := t.FieldByName("Node")
		nodeSpec := parseSpec(nodeField.Tag.Get("spec"))

		commands[toolName] = commandEntry{
			nodeType:    t,
			nodeID:      nodeSpec["id"],
			toolName:    toolName,
			description: toolParts["description"],
		}
	}

	return commands
}

// parseFlags parses --key=value and --key value style arguments.
// Keys are kept in kebab-case as provided.
func parseFlags(args []string) (map[string]string, error) {
	flags := make(map[string]string)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("unexpected argument %q (expected --flag=value)", arg)
		}

		arg = arg[2:] // strip --

		if idx := strings.IndexByte(arg, '='); idx >= 0 {
			// --key=value
			flags[arg[:idx]] = arg[idx+1:]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			// --key value
			flags[arg] = args[i+1]
			i++
		} else {
			// --flag (boolean true)
			flags[arg] = "true"
		}
	}

	return flags, nil
}

// buildMessageContext converts CLI flags into a message context map.
// Flag names (kebab-case) are converted to the spec tag "name" (camelCase) where possible.
func buildMessageContext(nodeType reflect.Type, flags map[string]string) map[string]interface{} {
	data := make(map[string]interface{})

	// Build a mapping from kebab-case flag name → spec tag name
	flagToSpec := buildFlagNameMap(nodeType)

	for flagName, value := range flags {
		// Convert flag name to spec name
		specName, ok := flagToSpec[flagName]
		if !ok {
			// Fall back: convert kebab-case to camelCase
			specName = strcase.ToLowerCamel(strings.ReplaceAll(flagName, "-", "_"))
		}
		data[specName] = value
	}

	return data
}

// buildFlagNameMap creates a mapping from kebab-case CLI flag names to the
// actual spec tag "name" values used in message context.
func buildFlagNameMap(nodeType reflect.Type) map[string]string {
	mapping := make(map[string]string)

	for i := 0; i < nodeType.NumField(); i++ {
		field := nodeType.Field(i)
		if !isVariable(field.Type) {
			continue
		}

		specTag := field.Tag.Get("spec")
		specMap := parseSpec(specTag)
		specName := specMap["name"]
		if specName == "" {
			continue
		}

		// Convert spec name to kebab-case for the CLI flag
		flagName := strcase.ToKebab(specName)
		mapping[flagName] = specName
	}

	return mapping
}

// injectCredentialConfig adds vault ID and item ID to the node config
// for any Credential fields found in the node type.
func injectCredentialConfig(nodeType reflect.Type, config map[string]interface{}, vaultID, itemID string) {
	for i := 0; i < nodeType.NumField(); i++ {
		field := nodeType.Field(i)
		if field.Type != reflect.TypeOf(Credential{}) {
			continue
		}

		fieldName := lowerFirstLetter(field.Name)
		config[fieldName] = map[string]interface{}{
			"scope": "Custom",
			"name": map[string]interface{}{
				"vaultId": vaultID,
				"itemId":  itemID,
			},
		}
	}
}

// collectCLIOutput reads output variable values from the message context after
// node execution. Returns a map suitable for JSON serialization.
func collectCLIOutput(nodeType reflect.Type, handler MessageHandler, ctx message.Context) map[string]interface{} {
	output := make(map[string]interface{})

	nodeValue := reflect.ValueOf(handler)
	if nodeValue.Kind() == reflect.Ptr {
		nodeValue = nodeValue.Elem()
	}

	for i := 0; i < nodeType.NumField(); i++ {
		field := nodeType.Field(i)
		if !isOutVariable(field.Type) {
			continue
		}

		specTag := field.Tag.Get("spec")
		specMap := parseSpec(specTag)
		name := specMap["name"]
		if name == "" {
			continue
		}

		if val := ctx.Get(name); val != nil {
			output[name] = val
		}
	}

	if len(output) == 0 {
		output["status"] = "completed"
	}

	return output
}

// listCommands prints all available CLI commands as JSON to stdout.
func listCommands() {
	RegisterFactories()

	types := GetNodeTypes()
	var commands []CLICommandInfo

	for _, t := range types {
		toolField, hasToolField := t.FieldByName("Tool")
		if !hasToolField {
			continue
		}

		toolTag := toolField.Tag.Get("tool")
		toolParts := parseSpec(toolTag)
		toolName := toolParts["name"]
		if toolName == "" {
			continue
		}

		nodeField, _ := t.FieldByName("Node")
		nodeSpec := parseSpec(nodeField.Tag.Get("spec"))

		cmd := CLICommandInfo{
			Name:        toolName,
			Description: toolParts["description"],
			NodeID:      nodeSpec["id"],
		}

		// Collect parameters and outputs
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			specTag := field.Tag.Get("spec")
			specMap := parseSpec(specTag)
			specName := specMap["name"]
			if specName == "" {
				continue
			}

			varType := specMap["type"]
			if varType == "" {
				varType = "string"
			}

			if isInVariable(field.Type) {
				cmd.Parameters = append(cmd.Parameters, CLIParamInfo{
					Name:        strcase.ToKebab(specName),
					Type:        varType,
					Required:    true,
					Description: specMap["description"],
				})
			} else if isOptVariable(field.Type) {
				cmd.Parameters = append(cmd.Parameters, CLIParamInfo{
					Name:        strcase.ToKebab(specName),
					Type:        varType,
					Required:    false,
					Description: specMap["description"],
				})
			} else if isOutVariable(field.Type) {
				cmd.Outputs = append(cmd.Outputs, CLIOutputInfo{
					Name: specName,
					Type: varType,
				})
			}
		}

		commands = append(commands, cmd)
	}

	data, _ := json.MarshalIndent(commands, "", "  ")
	fmt.Println(string(data))
}

// printCLIUsage prints human-readable help to stderr.
func printCLIUsage() {
	config := ReadConfigFile()
	name := config.Get("name").String()

	fmt.Fprintf(os.Stderr, "Usage: %s <command> [flags]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Package: %s\n\n", name)
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  --list-commands    List all available commands as JSON\n")
	fmt.Fprintf(os.Stderr, "  --skill-md         Generate SKILL.md for this package\n")
	fmt.Fprintf(os.Stderr, "  --help, -h         Show this help\n")
	fmt.Fprintf(os.Stderr, "  <command>          Run a command (use --list-commands to see available)\n")
	fmt.Fprintf(os.Stderr, "\nGlobal Flags:\n")
	fmt.Fprintf(os.Stderr, "  --vault-id=ID            Robomotion vault ID for credentials\n")
	fmt.Fprintf(os.Stderr, "  --item-id=ID             Robomotion vault item ID for credentials\n")
	fmt.Fprintf(os.Stderr, "\nSession Flags:\n")
	fmt.Fprintf(os.Stderr, "  --session                Start a new session (keeps process alive)\n")
	fmt.Fprintf(os.Stderr, "  --session-id=ID          Reuse an existing session\n")
	fmt.Fprintf(os.Stderr, "  --session-close=ID       Close a session and stop the daemon\n")
	fmt.Fprintf(os.Stderr, "  --session-timeout=DUR    Inactivity timeout (default 30m, e.g. 5m)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment:\n")
	fmt.Fprintf(os.Stderr, "  ROBOMOTION_CREDENTIALS   JSON credential map (alternative to vault flags)\n")
}

// cliError prints a JSON error to stderr and exits with code 1.
func cliError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	errJSON, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintln(os.Stderr, string(errJSON))
	os.Exit(1)
}
