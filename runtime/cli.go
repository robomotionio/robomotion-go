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
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
	Default     string   `json:"default,omitempty"`
	Choices     []string `json:"choices,omitempty"`
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
		// Handled by registry.go Start() — but if reached via RunCLI, dispatch
		if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
			listCommandDetail(args[1])
		} else {
			listCommands()
		}
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
	vaultName := flags["vault"]
	itemName := flags["item"]
	delete(flags, "vault-id")
	delete(flags, "item-id")
	delete(flags, "vault")
	delete(flags, "item")

	// Resolve --vault/--item names to IDs if needed
	if vaultName != "" || itemName != "" {
		if vaultID != "" || itemID != "" {
			cliError("cannot use --vault/--item together with --vault-id/--item-id")
			return
		}
		if vaultName == "" || itemName == "" {
			cliError("--vault and --item must both be provided")
			return
		}
		vc, err := NewCLIVaultClient()
		if err != nil {
			cliError("vault auth error: %v", err)
			return
		}
		vaultID, err = vc.ResolveVaultByName(vaultName)
		if err != nil {
			cliError("%v", err)
			return
		}
		itemID, err = vc.ResolveItemByName(vaultID, itemName)
		if err != nil {
			cliError("%v", err)
			return
		}
	}

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

	// Initialize all variable fields with scope/name config so they don't panic on Get/Set
	injectVariableConfig(cmd.nodeType, nodeConfig)

	// If vault flags provided, inject credential config for Credential fields
	if vaultID != "" && itemID != "" {
		injectCredentialConfig(cmd.nodeType, nodeConfig, vaultID, itemID)
	}

	// Build CLI context: split flags into message context vs config patches
	msgData, configPatches := buildCLIContext(cmd.nodeType, flags)

	// Apply config patches (Custom-scope variables, option fields)
	for k, v := range configPatches {
		nodeConfig[k] = v
	}

	configJSON, _ := json.Marshal(nodeConfig)
	if err := json.Unmarshal(configJSON, handler); err != nil {
		cliError("failed to initialize node: %v", err)
		return
	}

	// Build message context for Message-scope variables
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

// buildCLIContext processes CLI flags and returns:
// - msgData: message context map (for Message-scope variables)
// - configPatches: map of field name → value to inject into node config (for Custom-scope variables and options)
func buildCLIContext(nodeType reflect.Type, flags map[string]string) (msgData map[string]interface{}, configPatches map[string]interface{}) {
	msgData = make(map[string]interface{})
	configPatches = make(map[string]interface{})

	flagMap := buildFlagMap(nodeType)

	for flagName, value := range flags {
		entry, ok := flagMap[flagName]
		if !ok {
			// Fall back: treat as message context key
			specName := strcase.ToLowerCamel(strings.ReplaceAll(flagName, "-", "_"))
			msgData[specName] = value
			continue
		}

		if entry.isOption {
			// Enum option: inject into config as plain field
			configPatches[entry.fieldName] = value
		} else if entry.scope == "Custom" {
			// Custom-scope variable: set Name to value via config
			configPatches[entry.fieldName] = map[string]interface{}{
				"scope": "Custom",
				"name":  value,
			}
		} else {
			// Message-scope variable: put in message context
			msgData[entry.specName] = value
		}
	}

	return
}

// cliFlagEntry describes how a CLI flag maps to a node field.
type cliFlagEntry struct {
	specName  string // the spec "name" value (for Message-scope context keys)
	fieldName string // Go struct field name (lowered first letter for config JSON)
	scope     string // "Message" or "Custom"
	isOption  bool   // true for enum option fields (not variables)
}

// buildFlagMap creates a mapping from kebab-case CLI flag names to their
// field metadata, handling both named (Message-scope) and unnamed (Custom-scope) variables,
// as well as enum option fields.
func buildFlagMap(nodeType reflect.Type) map[string]cliFlagEntry {
	mapping := make(map[string]cliFlagEntry)

	for i := 0; i < nodeType.NumField(); i++ {
		field := nodeType.Field(i)
		specTag := field.Tag.Get("spec")
		specMap := parseSpec(specTag)

		if isVariable(field.Type) {
			specName := specMap["name"]
			scope := specMap["scope"]
			if scope == "" {
				scope = "Custom"
			}

			if specName != "" {
				// Named variable: flag from spec name
				flagName := strcase.ToKebab(specName)
				mapping[flagName] = cliFlagEntry{
					specName:  specName,
					fieldName: lowerFirstLetter(field.Name),
					scope:     scope,
				}
			} else {
				// Unnamed variable (Custom scope): derive flag from title
				title := specMap["title"]
				if title == "" {
					continue
				}
				flagName := strcase.ToKebab(title)
				mapping[flagName] = cliFlagEntry{
					fieldName: lowerFirstLetter(field.Name),
					scope:     "Custom",
				}
			}
		} else if enum := specMap["enum"]; enum != "" {
			// Enum option field
			title := specMap["title"]
			if title == "" {
				title = field.Name
			}
			flagName := strcase.ToKebab(title)
			mapping[flagName] = cliFlagEntry{
				fieldName: lowerFirstLetter(field.Name),
				isOption:  true,
			}
		}
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

// injectVariableConfig initializes all variable fields (In, Out, Opt) with their
// scope and name from spec tags so they don't panic on Get/Set during CLI execution.
func injectVariableConfig(nodeType reflect.Type, config map[string]interface{}) {
	for i := 0; i < nodeType.NumField(); i++ {
		field := nodeType.Field(i)
		if !isVariable(field.Type) {
			continue
		}

		specTag := field.Tag.Get("spec")
		specMap := parseSpec(specTag)
		specName := specMap["name"]
		scope := specMap["scope"]
		if scope == "" {
			scope = "Custom"
		}

		fieldKey := lowerFirstLetter(field.Name)

		// Don't overwrite if already set (e.g., by configPatches later)
		if _, exists := config[fieldKey]; exists {
			continue
		}

		if scope == "Message" && specName != "" {
			config[fieldKey] = map[string]interface{}{
				"scope": "Message",
				"name":  specName,
			}
		}
		// Custom-scope with empty name: will be set by configPatches from CLI flags
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

// gatherCommands collects CLI command metadata from all registered node types.
func gatherCommands() []CLICommandInfo {
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

			varType := specMap["type"]
			if varType == "" {
				varType = "string"
			}

			if isInVariable(field.Type) || isOptVariable(field.Type) {
				// Derive flag name: from spec name if present, else from title
				flagName := ""
				if specName != "" {
					flagName = strcase.ToKebab(specName)
				} else if title := specMap["title"]; title != "" {
					flagName = strcase.ToKebab(title)
				}
				if flagName == "" {
					continue
				}

				_, isOptional := specMap["optional"]
				required := isInVariable(field.Type) && !isOptional

				param := CLIParamInfo{
					Name:        flagName,
					Type:        varType,
					Required:    required,
					Description: specMap["description"],
				}

				// Include default value and enum for human-readable output
				if defVal := specMap["value"]; defVal != "" {
					param.Default = defVal
				}
				if enum := specMap["enum"]; enum != "" {
					param.Choices = strings.Split(enum, "|")
				}

				cmd.Parameters = append(cmd.Parameters, param)
			} else if isOutVariable(field.Type) {
				if specName == "" {
					continue
				}
				cmd.Outputs = append(cmd.Outputs, CLIOutputInfo{
					Name: specName,
					Type: varType,
				})
			} else if enum := specMap["enum"]; enum != "" {
				// Option fields (enums like OptStorage, OptEncoding)
				flagName := ""
				if title := specMap["title"]; title != "" {
					flagName = strcase.ToKebab(title)
				} else {
					flagName = strcase.ToKebab(field.Name)
				}
				param := CLIParamInfo{
					Name:        flagName,
					Type:        varType,
					Required:    false,
					Description: specMap["description"],
				}
				if defVal := specMap["value"]; defVal != "" {
					param.Default = defVal
				}
				param.Choices = strings.Split(enum, "|")
				cmd.Parameters = append(cmd.Parameters, param)
			}
		}

		commands = append(commands, cmd)
	}

	return commands
}

// listCommands prints available commands. Human-readable by default, JSON with --output json.
func listCommands() {
	// Check for --output json in remaining args
	outputJSON := false
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--output" && i+1 < len(os.Args) && os.Args[i+1] == "json" {
			outputJSON = true
			break
		}
		if os.Args[i] == "--output=json" {
			outputJSON = true
			break
		}
	}

	commands := gatherCommands()

	if outputJSON {
		data, _ := json.MarshalIndent(commands, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Human-readable output
	config := ReadConfigFile()
	name := config.Get("name").String()
	version := config.Get("version").String()

	fmt.Fprintf(os.Stderr, "%s v%s\n\n", name, version)
	fmt.Fprintf(os.Stderr, "Available commands:\n\n")

	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "  %-24s %s\n", cmd.Name, cmd.Description)
	}

	fmt.Fprintf(os.Stderr, "\nUse --list-commands <command> for details on a specific command.\n")
	fmt.Fprintf(os.Stderr, "Use --list-commands --output json for machine-readable output.\n")
}

// listCommandDetail prints detailed help for a single command.
func listCommandDetail(cmdName string) {
	commands := gatherCommands()

	var cmd *CLICommandInfo
	for i := range commands {
		if commands[i].Name == cmdName {
			cmd = &commands[i]
			break
		}
	}

	if cmd == nil {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmdName)
		fmt.Fprintf(os.Stderr, "Run --list-commands to see available commands.\n")
		os.Exit(1)
	}

	config := ReadConfigFile()
	binName := config.Get("name").String()

	fmt.Fprintf(os.Stderr, "Usage: %s %s [flags]\n\n", strings.ToLower(binName), cmd.Name)
	fmt.Fprintf(os.Stderr, "%s\n", cmd.Description)

	if len(cmd.Parameters) > 0 {
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		for _, p := range cmd.Parameters {
			flag := fmt.Sprintf("--%s %s", p.Name, p.Type)
			reqTag := ""
			if p.Required {
				reqTag = " (required)"
			}
			defTag := ""
			if p.Default != "" {
				defTag = fmt.Sprintf(" (default: %s)", p.Default)
			}
			choiceTag := ""
			if len(p.Choices) > 0 {
				choiceTag = fmt.Sprintf(" [%s]", strings.Join(p.Choices, "|"))
			}
			fmt.Fprintf(os.Stderr, "  %-32s %s%s%s%s\n", flag, p.Description, reqTag, defTag, choiceTag)
		}
	}

	if len(cmd.Outputs) > 0 {
		fmt.Fprintf(os.Stderr, "\nOutput:\n")
		for _, o := range cmd.Outputs {
			fmt.Fprintf(os.Stderr, "  %-32s %s\n", o.Name, o.Type)
		}
	}

	fmt.Fprintf(os.Stderr, "\nSession Flags:\n")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--session", "Start a new session (keeps process alive)")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--session-id ID", "Reuse an existing session")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--session-close ID", "Close a session")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--session-timeout DURATION", "Inactivity timeout (default: 30m)")
}

// printCLIUsage prints human-readable help to stderr.
func printCLIUsage() {
	config := ReadConfigFile()
	name := config.Get("name").String()
	version := config.Get("version").String()

	fmt.Fprintf(os.Stderr, "%s v%s\n\n", name, version)
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [flags]\n\n", strings.ToLower(name))

	fmt.Fprintf(os.Stderr, "Commands:\n")
	commands := gatherCommands()
	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "  %-24s %s\n", cmd.Name, cmd.Description)
	}

	fmt.Fprintf(os.Stderr, "\nGlobal Flags:\n")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--output json", "Output in JSON format")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--vault-id ID", "Robomotion vault ID for credentials")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--item-id ID", "Robomotion vault item ID for credentials")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--vault NAME", "Vault name (resolved to ID via API)")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--item NAME", "Item name (resolved to ID via API)")

	fmt.Fprintf(os.Stderr, "\nSession Flags:\n")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--session", "Start a new session (keeps process alive)")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--session-id ID", "Reuse an existing session")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--session-close ID", "Close a session and stop the daemon")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "--session-timeout DURATION", "Inactivity timeout (default: 30m)")

	fmt.Fprintf(os.Stderr, "\nEnvironment:\n")
	fmt.Fprintf(os.Stderr, "  %-32s %s\n", "ROBOMOTION_CREDENTIALS", "JSON credential map (alternative to vault flags)")

	fmt.Fprintf(os.Stderr, "\nUse --list-commands <command> for details on a specific command.\n")
	fmt.Fprintf(os.Stderr, "Use --help or -h to show this help.\n")
}

// cliError prints a JSON error to stderr and exits with code 1.
func cliError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	errJSON, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintln(os.Stderr, string(errJSON))
	os.Exit(1)
}
