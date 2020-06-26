package runtime

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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

func (c *Credentials) GetVaultItem() (map[string]interface{}, error) {
	if runtimeHelper == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetVaultItem(c.VaultID, c.ItemID)
}

func (variable *Variable) GetInteger(msg gjson.Result) (int32, error) {
	val, err := variable.GetValue(msg)
	if err != nil {
		return 0, err
	}

	if d, ok := val.(int32); ok {
		return d, nil
	}

	return 0, nil
}

func (variable *Variable) GetString(msg gjson.Result) (string, error) {
	val, err := variable.GetValue(msg)
	if err != nil {
		return "", err
	}

	if s, ok := val.(string); ok {
		return s, nil
	}

	return "", nil
}

func (variable *Variable) GetFloat(msg gjson.Result) (float32, error) {
	val, err := variable.GetValue(msg)
	if err != nil {
		return 0.0, err
	}

	if f, ok := val.(float32); ok {
		return f, nil
	}

	return 0.0, nil
}

func (variable *Variable) GetValue(msg gjson.Result) (interface{}, error) {
	if variable.Scope == "Message" {
		return msg.Get(variable.Name).Value(), nil
	}

	if runtimeHelper == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetVariable(variable)
}

func (variable *Variable) SetValue(msg *gjson.Result, value interface{}) error {
	if variable.Scope == "Message" {
		if variable.Name == "" {
			return fmt.Errorf("Empty message object")
		}

		sMsg, err := sjson.Set(msg.String(), variable.Name, value)
		if err != nil {
			return err
		}

		*msg = gjson.Parse(sMsg)
		return nil
	}

	if runtimeHelper == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.SetVariable(variable, value)
}
