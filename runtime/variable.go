package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/robomotionio/robomotion-go/message"
)

type variable struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

type Variable[T any] struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

type InVariable[T any] struct {
	Variable[T]
}

type OutVariable[T any] struct {
	Variable[T]
}

type OptVariable[T any] struct {
	InVariable[T]
}

func (v *InVariable[T]) getInt(val interface{}) (t T, err error) {

	switch v := val.(type) {
	case int64:
		reflect.ValueOf(&t).Elem().SetInt(v)
	case float64:
		reflect.ValueOf(&t).Elem().SetInt(int64(v))
	case string:
		var d int64
		d, err = strconv.ParseInt(v, 10, 64)
		if err == nil {
			reflect.ValueOf(&t).Elem().SetInt(d)
		}
	}

	return t, err
}

func (v *InVariable[T]) getFloat(val interface{}) (t T, err error) {
	switch v := val.(type) {
	case int64:
		reflect.ValueOf(&t).Elem().SetFloat(float64(v))
	case float64:
		reflect.ValueOf(&t).Elem().SetFloat(v)
	case string:
		var d float64
		d, err = strconv.ParseFloat(v, 64)
		if err == nil {
			reflect.ValueOf(&t).Elem().SetFloat(d)
		}
	}

	return t, err
}

func (v *InVariable[T]) getBool(val interface{}) (t T, err error) {
	switch v := val.(type) {
	case int64:
		reflect.ValueOf(&t).Elem().SetBool(v > 0)
	case float64:
		reflect.ValueOf(&t).Elem().SetBool(v > 0)
	case string:
		var d bool
		d, err = strconv.ParseBool(v)
		if err == nil {
			reflect.ValueOf(&t).Elem().SetBool(d)
		}
	}

	return t, err
}

func (v *InVariable[T]) getString(val interface{}) (t T, err error) {
	switch v := val.(type) {
	case string:
		reflect.ValueOf(&t).Elem().SetString(v)
	}

	return t, err
}

func (v *InVariable[T]) Get(ctx message.Context) (T, error) {
	var (
		t   T
		val interface{}
	)

	if v.Scope == "Custom" {
		val = v.Name
	}

	if v.Scope == "Message" {
		val = ctx.Get(v.Name)
	}

	kind := reflect.Invalid
	typ := reflect.TypeOf(t)
	if typ != nil {
		kind = typ.Kind()
	}

	if val != nil {
		switch kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return v.getInt(val)

		case reflect.Float32, reflect.Float64:
			return v.getFloat(val)

		case reflect.Bool:
			return v.getBool(val)

		case reflect.String:
			return v.getString(val)

		default:
			d, err := json.Marshal(val)
			if err != nil {
				return t, err
			}

			err = json.Unmarshal(d, &t)
			if err != nil {
				return t, err
			}

			return t, nil
		}
	}

	if client == nil {
		return t, fmt.Errorf("Runtime was not initialized")
	}

	val, err := client.GetVariable(&variable{Scope: v.Scope, Name: v.Name})
	if err != nil {
		return t, err
	}

	t, ok := val.(T)
	if !ok {
		return t, fmt.Errorf("expected %s but got %s",
			reflect.TypeOf(t).String(),
			reflect.TypeOf(val).String(),
		)
	}

	return t, nil
}

func (v *OutVariable[T]) Set(ctx message.Context, value T) error {

	if v.Scope == "Message" {
		if v.Name == "" {
			return fmt.Errorf("Empty message object")
		}

		return ctx.Set(v.Name, value)
	}

	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.SetVariable(&variable{Scope: v.Scope, Name: v.Name}, value)
}
