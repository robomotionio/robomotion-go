package runtime

import (
	"fmt"
	"strconv"

	"github.com/robomotionio/robomotion-go/message"
)

type Variable struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

type InVariable struct {
	Variable
}

type OutVariable struct {
	Variable
}

type OptVariable struct {
	InVariable
}

func (v *InVariable) GetInt(ctx message.Context) (int64, error) {
	val, err := v.Get(ctx)
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, nil
	}
}

func (v *InVariable) GetString(ctx message.Context) (string, error) {
	val, err := v.Get(ctx)
	if err != nil {
		return "", err
	}

	if s, ok := val.(string); ok {
		return s, nil
	}

	return "", nil
}

func (v *InVariable) GetFloat(ctx message.Context) (float64, error) {
	val, err := v.Get(ctx)
	if err != nil {
		return 0.0, err
	}

	if f, ok := val.(float64); ok {
		return f, nil
	}

	return 0.0, nil
}

func (v *InVariable) GetBool(ctx message.Context) (bool, error) {
	val, err := v.Get(ctx)
	if err != nil {
		return false, err
	}

	if f, ok := val.(bool); ok {
		return f, nil
	}

	return false, nil
}

func (v *InVariable) Get(ctx message.Context) (interface{}, error) {

	if v.Scope == "Message" {
		return ctx.Get(v.Name), nil
	}

	if client == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return client.GetVariable(&v.Variable)
}

func (v *OutVariable) Set(ctx message.Context, value interface{}) error {

	if v.Scope == "Message" {
		if v.Name == "" {
			return fmt.Errorf("Empty message object")
		}

		return ctx.Set(v.Name, value)
	}

	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.SetVariable(&v.Variable, value)
}
