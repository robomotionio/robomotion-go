package runtime

import "fmt"

func EmitDebug(guid, name string, message interface{}) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.Debug(guid, name, message)
}
