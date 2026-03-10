package runtime

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/tidwall/gjson"
)

// generateSkillMD introspects registered nodes and prints a SKILL.md to stdout.
// Only nodes with an embedded Tool field are included.
func generateSkillMD(name, version string, config gjson.Result) {
	RegisterFactories()

	description := config.Get("description").String()
	if description == "" {
		description = name
	}

	binaryName := inferBinaryName(name)
	types := GetNodeTypes()

	var commands []skillCommand
	var hasCredentials bool

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

		cmd := skillCommand{
			name:        toolName,
			description: toolParts["description"],
		}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			if field.Type == reflect.TypeOf(Credential{}) {
				hasCredentials = true
				continue
			}

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

			flagName := strcase.ToKebab(specName)
			desc := specMap["description"]
			title := specMap["title"]
			if desc == "" && title != "" {
				desc = title
			}

			if isInVariable(field.Type) {
				cmd.params = append(cmd.params, skillParam{
					flag:        flagName,
					varType:     varType,
					required:    true,
					description: desc,
				})
			} else if isOptVariable(field.Type) {
				cmd.params = append(cmd.params, skillParam{
					flag:        flagName,
					varType:     varType,
					required:    false,
					description: desc,
				})
			} else if isOutVariable(field.Type) {
				cmd.outputs = append(cmd.outputs, skillOutput{
					name:    specName,
					varType: varType,
				})
			}
		}

		commands = append(commands, cmd)
	}

	if len(commands) == 0 {
		fmt.Fprintf(os.Stderr, "No tool-enabled nodes found in package %s\n", name)
		os.Exit(1)
	}

	// Generate SKILL.md
	var b strings.Builder

	// YAML frontmatter
	b.WriteString("---\n")
	fmt.Fprintf(&b, "name: %s\n", strings.ToLower(strings.TrimPrefix(name, "Robomotion.")))
	fmt.Fprintf(&b, "description: %s\n", description)
	b.WriteString("---\n\n")

	// Title
	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintf(&b, "Binary: `%s`\n\n", binaryName)

	// Commands section
	b.WriteString("## Commands\n\n")

	for _, cmd := range commands {
		fmt.Fprintf(&b, "### %s\n", cmd.name)
		if cmd.description != "" {
			fmt.Fprintf(&b, "%s\n", cmd.description)
		}
		b.WriteString("\n")

		// Usage line
		fmt.Fprintf(&b, "    %s %s", binaryName, cmd.name)
		for _, p := range cmd.params {
			if p.required {
				fmt.Fprintf(&b, " --%s=<%s>", p.flag, p.varType)
			}
		}
		for _, p := range cmd.params {
			if !p.required {
				fmt.Fprintf(&b, " [--%s=<%s>]", p.flag, p.varType)
			}
		}
		b.WriteString("\n\n")

		// Parameters
		if len(cmd.params) > 0 {
			b.WriteString("**Parameters:**\n")
			for _, p := range cmd.params {
				req := "optional"
				if p.required {
					req = "required"
				}
				desc := ""
				if p.description != "" {
					desc = ": " + p.description
				}
				fmt.Fprintf(&b, "- `--%s` (%s, %s)%s\n", p.flag, p.varType, req, desc)
			}
			b.WriteString("\n")
		}

		// Output
		if len(cmd.outputs) > 0 {
			outputFields := make([]string, 0, len(cmd.outputs))
			for _, o := range cmd.outputs {
				outputFields = append(outputFields, fmt.Sprintf(`"%s": "..."`, o.name))
			}
			fmt.Fprintf(&b, "**Output:** `{%s}`\n\n", strings.Join(outputFields, ", "))
		}
	}

	// Authentication section
	if hasCredentials {
		b.WriteString("## Authentication\n\n")
		b.WriteString("All commands require Robomotion Vault credentials via `--vault-id` and `--item-id` flags.\n")
		b.WriteString("Each call is stateless — parallel calls with different credentials are fully isolated.\n\n")
		fmt.Fprintf(&b, "    %s %s --vault-id=<id> --item-id=<id> ...\n\n",
			binaryName, commands[0].name)
		b.WriteString("Alternatively, set the `ROBOMOTION_CREDENTIALS` environment variable to a JSON credential map.\n")
	}

	fmt.Print(b.String())
}

// inferBinaryName derives the binary name from the package name.
// e.g., "Robomotion.GoogleDrive" → "robomotion-googledrive"
func inferBinaryName(pkgName string) string {
	// Remove namespace prefix and convert to lowercase
	parts := strings.Split(pkgName, ".")
	if len(parts) > 1 {
		return "robomotion-" + strings.ToLower(parts[len(parts)-1])
	}
	return "robomotion-" + strings.ToLower(pkgName)
}

type skillCommand struct {
	name        string
	description string
	params      []skillParam
	outputs     []skillOutput
}

type skillParam struct {
	flag        string
	varType     string
	required    bool
	description string
}

type skillOutput struct {
	name    string
	varType string
}
