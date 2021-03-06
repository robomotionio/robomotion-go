package runtime

import "fmt"

func EmitDebug(guid, name string, message interface{}) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.Debug(guid, name, message)
}
func EmitOutput(guid string, output []byte, port int32) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.EmitOutput(guid, output, port)
}
func EmitInput(guid string, input []byte) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.EmitInput(guid, input)
}
func EmitError(guid, name, message string) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.EmitError(guid, name, message)
}
func EmitFlowEvent(guid, name string) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.EmitFlowEvent(guid, name)
}
