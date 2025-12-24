package testing

import (
	"reflect"
)

// Quick provides a simplified interface for common testing patterns.
// It automatically configures variables based on spec tags when possible.
type Quick struct {
	harness *Harness
	node    interface{}
}

// NewQuick creates a Quick test helper that auto-configures node variables.
// It parses spec tags from the node struct to configure variables automatically.
//
// Example:
//
//	q := testing.NewQuick(&object.AddProperty{}).
//	    SetInput("obj", map[string]interface{}{"foo": "bar"}).
//	    SetInput("val", "hello").
//	    SetCustom("InKey", "mykey")
//
//	err := q.Run()
//	result := q.Output("obj")
func NewQuick(node MessageHandler) *Quick {
	q := &Quick{
		harness: NewHarness(node),
		node:    node,
	}
	q.autoConfigureVariables()
	return q
}

// autoConfigureVariables parses spec tags and configures variables automatically.
func (q *Quick) autoConfigureVariables() {
	nodeVal := reflect.ValueOf(q.node).Elem()
	nodeType := nodeVal.Type()

	for i := 0; i < nodeType.NumField(); i++ {
		field := nodeType.Field(i)
		fieldVal := nodeVal.Field(i)

		// Skip non-exported or non-struct fields
		if !fieldVal.CanAddr() || fieldVal.Kind() != reflect.Struct {
			continue
		}

		// Check if this is a Variable type (InVariable, OutVariable, OptVariable)
		if !isVariableType(field.Type) {
			continue
		}

		// Parse spec tag
		spec := field.Tag.Get("spec")
		if spec == "" {
			continue
		}

		specMap := parseSpec(spec)

		// Get scope and name from spec
		scope := specMap["scope"]
		name := specMap["name"]

		if scope == "" {
			scope = "Message" // Default scope
		}

		if name == "" {
			// For variables without a name (like OptVariable with empty name=),
			// configure them with Custom scope and empty string value.
			// This prevents panics when Get() is called on unconfigured variables.
			configureVariableCustom(fieldVal.Addr().Interface(), "")
			continue
		}

		// Configure the variable
		configureVariable(fieldVal.Addr().Interface(), scope, name)
	}
}

// configureVariableCustom sets a variable to Custom scope with a given value.
func configureVariableCustom(variable interface{}, value interface{}) {
	v := reflect.ValueOf(variable).Elem()

	// Find the embedded Variable struct
	var varField reflect.Value
	if v.Kind() == reflect.Struct {
		if f := v.FieldByName("Variable"); f.IsValid() {
			varField = f
		} else if f := v.FieldByName("InVariable"); f.IsValid() {
			varField = f.FieldByName("Variable")
		}
	}

	if !varField.IsValid() {
		varField = v
	}

	// Set Scope to "Custom"
	if scopeField := varField.FieldByName("Scope"); scopeField.IsValid() && scopeField.CanSet() {
		scopeField.SetString("Custom")
	}

	// Set Name to the value (for Custom scope, Name holds the value)
	if nameField := varField.FieldByName("Name"); nameField.IsValid() && nameField.CanSet() {
		nameField.Set(reflect.ValueOf(value))
	}
}

// SetInput sets an input value in the message context.
func (q *Quick) SetInput(name string, value interface{}) *Quick {
	q.harness.WithInput(name, value)
	return q
}

// SetInputs sets multiple input values.
func (q *Quick) SetInputs(inputs map[string]interface{}) *Quick {
	q.harness.WithInputs(inputs)
	return q
}

// SetCustom sets a Custom scope value for a field by field name.
// This is useful for configuring options that don't come from the message.
//
// Example:
//
//	q.SetCustom("InKey", "propertyName")
func (q *Quick) SetCustom(fieldName string, value interface{}) *Quick {
	nodeVal := reflect.ValueOf(q.node).Elem()
	field := nodeVal.FieldByName(fieldName)

	if field.IsValid() && field.CanAddr() {
		q.harness.ConfigureCustomInput(field.Addr().Interface(), value)
	}

	return q
}

// ConfigureVariable allows manual configuration of a variable by field name.
func (q *Quick) ConfigureVariable(fieldName, scope, name string) *Quick {
	nodeVal := reflect.ValueOf(q.node).Elem()
	field := nodeVal.FieldByName(fieldName)

	if field.IsValid() && field.CanAddr() {
		configureVariable(field.Addr().Interface(), scope, name)
	}

	return q
}

// Run executes the node's OnMessage method.
func (q *Quick) Run() error {
	return q.harness.Run()
}

// RunFull runs the complete lifecycle: OnCreate, OnMessage, OnClose.
func (q *Quick) RunFull() error {
	return q.harness.RunFull()
}

// Output retrieves an output value from the context.
func (q *Quick) Output(name string) interface{} {
	return q.harness.GetOutput(name)
}

// OutputString retrieves an output as a string.
func (q *Quick) OutputString(name string) string {
	return q.harness.GetOutputString(name)
}

// OutputInt retrieves an output as an int64.
func (q *Quick) OutputInt(name string) int64 {
	return q.harness.GetOutputInt(name)
}

// OutputFloat retrieves an output as a float64.
func (q *Quick) OutputFloat(name string) float64 {
	return q.harness.GetOutputFloat(name)
}

// OutputBool retrieves an output as a bool.
func (q *Quick) OutputBool(name string) bool {
	return q.harness.GetOutputBool(name)
}

// Outputs returns all output values.
func (q *Quick) Outputs() map[string]interface{} {
	return q.harness.GetAllOutputs()
}

// Harness returns the underlying Harness for advanced usage.
func (q *Quick) Harness() *Harness {
	return q.harness
}

// Context returns the underlying MockContext.
func (q *Quick) Context() *MockContext {
	return q.harness.Context()
}

// GetOutput is an alias for Output for convenience.
func (q *Quick) GetOutput(name string) interface{} {
	return q.harness.GetOutput(name)
}

// SetCredential configures a Credential field to use a mock credential.
// The fieldName is the name of the Credential field on the node struct.
// The vaultID and itemID are used to look up the credential from the CredentialStore.
//
// Example:
//
//	// First set up credentials in TestMain:
//	store := testing.NewCredentialStore()
//	store.SetAPIKey("my_api_key", "secret123")
//	testing.InitCredentials(store)
//
//	// Then in your test:
//	q.SetCredential("OptApiKey", "my_api_key", "")
func (q *Quick) SetCredential(fieldName, vaultID, itemID string) *Quick {
	nodeVal := reflect.ValueOf(q.node).Elem()
	field := nodeVal.FieldByName(fieldName)

	if field.IsValid() && field.CanSet() {
		// Get the VaultID field within the Credential struct
		vaultField := field.FieldByName("VaultID")
		if vaultField.IsValid() && vaultField.CanSet() {
			vaultField.SetString(vaultID)
		}
		// Get the ItemID field within the Credential struct
		itemField := field.FieldByName("ItemID")
		if itemField.IsValid() && itemField.CanSet() {
			itemField.SetString(itemID)
		}
	}

	return q
}

// Reset clears the context for reuse.
func (q *Quick) Reset() *Quick {
	q.harness.Reset()
	return q
}

// isVariableType checks if the type is an InVariable, OutVariable, or OptVariable.
func isVariableType(t reflect.Type) bool {
	name := t.Name()
	return len(name) > 0 &&
		(contains(name, "InVariable") ||
			contains(name, "OutVariable") ||
			contains(name, "OptVariable"))
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr) >= 0
}

// searchString finds substr in s.
func searchString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// parseSpec parses a spec tag into a map of key-value pairs.
func parseSpec(spec string) map[string]string {
	result := make(map[string]string)

	i := 0
	for i < len(spec) {
		// Skip leading whitespace and commas
		for i < len(spec) && (spec[i] == ' ' || spec[i] == ',') {
			i++
		}
		if i >= len(spec) {
			break
		}

		// Find key
		keyStart := i
		for i < len(spec) && spec[i] != '=' && spec[i] != ',' {
			i++
		}
		key := spec[keyStart:i]

		if i >= len(spec) || spec[i] == ',' {
			// Key without value (flag)
			result[key] = ""
			continue
		}

		// Skip '='
		i++
		if i >= len(spec) {
			result[key] = ""
			break
		}

		// Find value
		valueStart := i
		for i < len(spec) && spec[i] != ',' {
			i++
		}
		value := spec[valueStart:i]
		result[key] = value
	}

	return result
}
