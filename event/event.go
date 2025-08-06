package event

import "github.com/robomotionio/robomotion-go/message"

// emitInputFunc is set by the runtime package to avoid import cycles
var emitInputFunc func(string, message.Context) error

// SetEmitInputFunc allows the runtime to register its EmitInput implementation
func SetEmitInputFunc(f func(string, message.Context) error) {
	emitInputFunc = f
}

// EmitInput sends a message to a node's input
func EmitInput(nodeID string, ctx message.Context) error {
	if emitInputFunc != nil {
		return emitInputFunc(nodeID, ctx)
	}
	// Fallback - no implementation registered
	return nil
}