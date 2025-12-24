// Package testing provides utilities for unit testing Robomotion nodes.
//
// This package allows testing of node OnMessage methods without requiring
// the full Robomotion runtime, GRPC, or NATS infrastructure.
//
// Example usage:
//
//	func TestAddProperty(t *testing.T) {
//	    node := &object.AddProperty{}
//	    h := testing.NewHarness(node).
//	        WithInput("obj", map[string]interface{}{"foo": "bar"}).
//	        WithInput("val", "hello").
//	        ConfigureVariable(&node.InKey, "Message", "mykey")
//
//	    err := h.Run()
//	    require.NoError(t, err)
//
//	    result := h.GetOutput("obj")
//	    assert.Equal(t, "hello", result.(map[string]interface{})["mykey"])
//	}
package testing

import (
	"encoding/json"
	"sync"

	"github.com/robomotionio/robomotion-go/message"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// MockContext implements message.Context for testing purposes.
// It stores values in memory without requiring the full Robomotion runtime.
type MockContext struct {
	mu   sync.RWMutex
	data []byte
	id   string
}

// NewMockContext creates a new MockContext with optional initial data.
// If no data is provided, an empty JSON object is used.
func NewMockContext(initialData ...map[string]interface{}) *MockContext {
	ctx := &MockContext{
		data: []byte("{}"),
	}

	if len(initialData) > 0 && initialData[0] != nil {
		data, err := json.Marshal(initialData[0])
		if err == nil {
			ctx.data = data
		}
		// Extract ID if present
		if id, ok := initialData[0]["id"].(string); ok {
			ctx.id = id
		}
	}

	return ctx
}

// NewMockContextFromJSON creates a MockContext from raw JSON bytes.
func NewMockContextFromJSON(jsonData []byte) *MockContext {
	ctx := &MockContext{
		data: jsonData,
		id:   gjson.GetBytes(jsonData, "id").String(),
	}
	return ctx
}

// GetID returns the message ID from the "id" field.
func (m *MockContext) GetID() string {
	return m.id
}

// Set sets a value at the given path using JSONPath notation.
func (m *MockContext) Set(path string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path = convertPath(path)
	var err error
	m.data, err = sjson.SetBytes(m.data, path, value)
	return err
}

// Get retrieves a value from the given path.
func (m *MockContext) Get(path string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path = convertPath(path)
	return gjson.GetBytes(m.data, path).Value()
}

// GetString retrieves a string value from the given path.
func (m *MockContext) GetString(path string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path = convertPath(path)
	return gjson.GetBytes(m.data, path).String()
}

// GetBool retrieves a boolean value from the given path.
func (m *MockContext) GetBool(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path = convertPath(path)
	return gjson.GetBytes(m.data, path).Bool()
}

// GetInt retrieves an integer value from the given path.
func (m *MockContext) GetInt(path string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path = convertPath(path)
	return gjson.GetBytes(m.data, path).Int()
}

// GetFloat retrieves a float value from the given path.
func (m *MockContext) GetFloat(path string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path = convertPath(path)
	return gjson.GetBytes(m.data, path).Float()
}

// GetRaw returns the raw JSON bytes.
func (m *MockContext) GetRaw(options ...message.GetOption) (json.RawMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := make([]byte, len(m.data))
	copy(data, m.data)

	for _, opt := range options {
		var err error
		data, err = opt(data)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// SetRaw sets the raw JSON bytes.
func (m *MockContext) SetRaw(data json.RawMessage, options ...message.SetOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, opt := range options {
		var err error
		data, err = opt(data)
		if err != nil {
			return err
		}
	}

	m.data = data
	return nil
}

// IsEmpty returns true if the context has no data.
func (m *MockContext) IsEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.data == nil || len(m.data) == 0
}

// GetAll returns all data as a map.
func (m *MockContext) GetAll() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result map[string]interface{}
	json.Unmarshal(m.data, &result)
	return result
}

// GetJSON returns the current state as JSON bytes.
func (m *MockContext) GetJSON() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := make([]byte, len(m.data))
	copy(data, m.data)
	return data
}

// convertPath converts bracket notation to dot notation for gjson/sjson.
func convertPath(path string) string {
	result := make([]byte, 0, len(path))
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '[':
			result = append(result, '.')
		case ']':
			// skip
		default:
			result = append(result, path[i])
		}
	}
	return string(result)
}
