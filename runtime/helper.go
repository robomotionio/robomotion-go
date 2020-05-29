package runtime

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

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

func (variable *Variable) GetInteger(message []byte) (int32, error) {
	if variable.Scope == "Message" {
		return int32(gjson.Get(string(message), variable.Name).Int()), nil
	}

	if runtimeHelper == nil {
		return 0, fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetIntVariable(variable, message)
}

func (variable *Variable) GetString(message []byte) (string, error) {
	if variable.Scope == "Message" {
		return gjson.Get(string(message), variable.Name).String(), nil
	}

	if runtimeHelper == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetStringVariable(variable, message)
}

func (variable *Variable) GetInterface(message []byte) (interface{}, error) {
	if variable.Scope == "Message" {
		return gjson.Get(string(message), variable.Name).Value(), nil
	}

	if runtimeHelper == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetInterfaceVariable(variable, message)
}

func (variable *Variable) SetInteger(msg map[string]interface{}, value int32) error {
	if variable.Scope == "Message" {
		if variable.Name == "" {
			return fmt.Errorf("Empty message object")
		}

		msg[variable.Name] = value
		return nil
	}

	if runtimeHelper == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	message, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = runtimeHelper.SetIntVariable(variable, message, value)
	return err
}

func (variable *Variable) SetString(msg map[string]interface{}, value string) error {
	if variable.Scope == "Message" {
		if variable.Name == "" {
			return fmt.Errorf("Empty message object")
		}

		msg[variable.Name] = value
		return nil
	}

	if runtimeHelper == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	message, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = runtimeHelper.SetStringVariable(variable, message, value)
	return err
}

func (variable *Variable) SetInterface(msg map[string]interface{}, value interface{}) error {
	if variable.Scope == "Message" {
		if variable.Name == "" {
			return fmt.Errorf("Empty message object")
		}

		msg[variable.Name] = value
		return nil
	}

	if runtimeHelper == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	message, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = runtimeHelper.SetInterfaceVariable(variable, message, value)
	return err
}
