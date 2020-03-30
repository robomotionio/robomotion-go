package runtime

import "fmt"

var (
	runtimeHelper RuntimeHelper
)

func Close() error {
	return runtimeHelper.Close()
}

func EmitDebug(guid, name string, message interface{}) error {
	if runtimeHelper == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.Debug(guid, name, message)
}

func GetVaultItem(vaultID, itemID string) (map[string]interface{}, error) {
	if runtimeHelper == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetVaultItem(vaultID, itemID)
}

func GetIntVariable(variable *Variable, message []byte) (int32, error) {
	if runtimeHelper == nil {
		return 0, fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetIntVariable(variable, message)
}

func GetStringVariable(variable *Variable, message []byte) (string, error) {
	if runtimeHelper == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetStringVariable(variable, message)
}

func GetInterfaceVariable(variable *Variable, message []byte) (interface{}, error) {
	if runtimeHelper == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetInterfaceVariable(variable, message)
}

func SetIntVariable(variable *Variable, message []byte, value int32) ([]byte, error) {
	if runtimeHelper == nil {
		return message, fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.SetIntVariable(variable, message, value)
}

func SetStringVariable(variable *Variable, message []byte, value string) ([]byte, error) {
	if runtimeHelper == nil {
		return message, fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.SetStringVariable(variable, message, value)
}

func SetInterfaceVariable(variable *Variable, message []byte, value interface{}) ([]byte, error) {
	if runtimeHelper == nil {
		return message, fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.SetInterfaceVariable(variable, message, value)
}
