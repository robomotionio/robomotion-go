package runtime

import (
	"fmt"

	"bitbucket.org/mosteknoloji/robomotion-go-lib/message"
)

type Variable struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

func (v *Variable) GetInt(ctx message.Context) (int64, error) {
	val, err := v.Get(ctx)
	if err != nil {
		return 0, err
	}

	if d, ok := val.(int64); ok {
		return d, nil
	}

	return 0, nil
}

func (v *Variable) GetString(ctx message.Context) (string, error) {
	val, err := v.Get(ctx)
	if err != nil {
		return "", err
	}

	if s, ok := val.(string); ok {
		return s, nil
	}

	return "", nil
}

func (v *Variable) GetFloat(ctx message.Context) (float64, error) {
	val, err := v.Get(ctx)
	if err != nil {
		return 0.0, err
	}

	if f, ok := val.(float64); ok {
		return f, nil
	}

	return 0.0, nil
}

func (v *Variable) GetBool(ctx message.Context) (bool, error) {
	val, err := v.Get(ctx)
	if err != nil {
		return false, err
	}

	if f, ok := val.(bool); ok {
		return f, nil
	}

	return false, nil
}

func (v *Variable) Get(ctx message.Context) (interface{}, error) {

	if v.Scope == "Message" {
		return ctx.Get(v.Name), nil
	}

	if client == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return client.GetVariable(v)
}

func (v *Variable) Set(ctx message.Context, value interface{}) error {

	if v.Scope == "Message" {
		if v.Name == "" {
			return fmt.Errorf("Empty message object")
		}

		return ctx.Set(v.Name, value)
	}

	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.SetVariable(v, value)
}
