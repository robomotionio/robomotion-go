package testing

import (
	"reflect"

	"github.com/robomotionio/robomotion-go/message"
)

// MessageHandler is the interface that all Robomotion nodes implement.
type MessageHandler interface {
	OnCreate() error
	OnMessage(ctx message.Context) error
	OnClose() error
}

// Harness provides a testing harness for Robomotion nodes.
// It allows setting up input values, configuring variables, running the node,
// and asserting on output values.
type Harness struct {
	node    MessageHandler
	ctx     *MockContext
	inputs  map[string]interface{}
	created bool
}

// NewHarness creates a new test harness for the given node.
func NewHarness(node MessageHandler) *Harness {
	return &Harness{
		node:   node,
		ctx:    NewMockContext(),
		inputs: make(map[string]interface{}),
	}
}

// WithInput sets an input value in the message context.
// The name should match the variable name configured for InVariable fields.
func (h *Harness) WithInput(name string, value interface{}) *Harness {
	h.inputs[name] = value
	h.ctx.Set(name, value)
	return h
}

// WithInputs sets multiple input values at once.
func (h *Harness) WithInputs(inputs map[string]interface{}) *Harness {
	for name, value := range inputs {
		h.WithInput(name, value)
	}
	return h
}

// WithContext sets a custom MockContext for the test.
func (h *Harness) WithContext(ctx *MockContext) *Harness {
	h.ctx = ctx
	return h
}

// Context returns the underlying MockContext.
func (h *Harness) Context() *MockContext {
	return h.ctx
}

// ConfigureInVariable configures an InVariable field with the given scope and name.
// This uses reflection to set the Scope and Name fields on the variable.
//
// Example:
//
//	h.ConfigureInVariable(&node.InObject, "Message", "obj")
func (h *Harness) ConfigureInVariable(variable interface{}, scope, name string) *Harness {
	configureVariable(variable, scope, name)
	return h
}

// ConfigureOutVariable configures an OutVariable field with the given scope and name.
//
// Example:
//
//	h.ConfigureOutVariable(&node.OutResult, "Message", "result")
func (h *Harness) ConfigureOutVariable(variable interface{}, scope, name string) *Harness {
	configureVariable(variable, scope, name)
	return h
}

// ConfigureCustomInput configures an InVariable with Custom scope and a direct value.
// This is useful for testing node options/parameters that don't come from the message.
//
// Example:
//
//	h.ConfigureCustomInput(&node.InKey, "myPropertyKey")
func (h *Harness) ConfigureCustomInput(variable interface{}, value interface{}) *Harness {
	v := reflect.ValueOf(variable).Elem()

	// Find the embedded Variable struct
	var varField reflect.Value
	if v.Kind() == reflect.Struct {
		// Check if it has a Variable or InVariable embedded
		if f := v.FieldByName("Variable"); f.IsValid() {
			varField = f
		} else if f := v.FieldByName("InVariable"); f.IsValid() {
			// For OptVariable which embeds InVariable
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

	// Normalize numeric types to match runtime expectations
	// The runtime's getInt() expects int64/float64, not int
	normalizedValue := normalizeNumericValue(value)

	// Set Name to the actual value (for Custom scope, Name holds the value)
	if nameField := varField.FieldByName("Name"); nameField.IsValid() && nameField.CanSet() {
		nameField.Set(reflect.ValueOf(normalizedValue))
	}

	return h
}

// normalizeNumericValue converts Go numeric types to the types expected by runtime.
// The runtime's getInt() handles int64 and float64, not int/int32/etc.
func normalizeNumericValue(value interface{}) interface{} {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float32:
		return float64(v)
	default:
		return value
	}
}

// Run executes the node's OnMessage method with the configured context.
// It returns any error from OnMessage.
func (h *Harness) Run() error {
	if !h.created {
		if err := h.node.OnCreate(); err != nil {
			return err
		}
		h.created = true
	}
	return h.node.OnMessage(h.ctx)
}

// RunWithCreate explicitly calls OnCreate before OnMessage.
func (h *Harness) RunWithCreate() error {
	if err := h.node.OnCreate(); err != nil {
		return err
	}
	h.created = true
	return h.node.OnMessage(h.ctx)
}

// RunFull runs the complete node lifecycle: OnCreate, OnMessage, OnClose.
func (h *Harness) RunFull() error {
	if err := h.node.OnCreate(); err != nil {
		return err
	}
	if err := h.node.OnMessage(h.ctx); err != nil {
		return err
	}
	return h.node.OnClose()
}

// GetOutput retrieves an output value from the message context.
func (h *Harness) GetOutput(name string) interface{} {
	return h.ctx.Get(name)
}

// GetOutputString retrieves an output value as a string.
func (h *Harness) GetOutputString(name string) string {
	return h.ctx.GetString(name)
}

// GetOutputInt retrieves an output value as an int64.
func (h *Harness) GetOutputInt(name string) int64 {
	return h.ctx.GetInt(name)
}

// GetOutputFloat retrieves an output value as a float64.
func (h *Harness) GetOutputFloat(name string) float64 {
	return h.ctx.GetFloat(name)
}

// GetOutputBool retrieves an output value as a bool.
func (h *Harness) GetOutputBool(name string) bool {
	return h.ctx.GetBool(name)
}

// GetAllOutputs returns all values in the context as a map.
func (h *Harness) GetAllOutputs() map[string]interface{} {
	return h.ctx.GetAll()
}

// Reset clears the context and input values for reuse.
func (h *Harness) Reset() *Harness {
	h.ctx = NewMockContext()
	h.inputs = make(map[string]interface{})
	return h
}

// configureVariable sets the Scope and Name fields on a variable using reflection.
func configureVariable(variable interface{}, scope, name string) {
	v := reflect.ValueOf(variable).Elem()

	// Find the embedded Variable struct
	var varField reflect.Value
	if v.Kind() == reflect.Struct {
		// Check if it has a Variable or InVariable embedded
		if f := v.FieldByName("Variable"); f.IsValid() {
			varField = f
		} else if f := v.FieldByName("InVariable"); f.IsValid() {
			// For OptVariable which embeds InVariable
			varField = f.FieldByName("Variable")
		}
	}

	if !varField.IsValid() {
		varField = v
	}

	// Set Scope
	if scopeField := varField.FieldByName("Scope"); scopeField.IsValid() && scopeField.CanSet() {
		scopeField.SetString(scope)
	}

	// Set Name
	if nameField := varField.FieldByName("Name"); nameField.IsValid() && nameField.CanSet() {
		nameField.Set(reflect.ValueOf(name))
	}
}
