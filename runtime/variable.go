package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	robocapnp "github.com/robomotionio/robomotion-go/capnp"
	"github.com/robomotionio/robomotion-go/message"
)

type variable struct {
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Payload []byte `json:"payload,omitempty"`
}

type Variable[T any] struct {
	Scope string      `json:"scope"`
	Name  interface{} `json:"name"`
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

func (v *Variable[T]) IsNil() bool {
	return v.Name == nil
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
	case map[string]interface{}:
		for _, val := range v {
			dval, ok := val.(int64)
			if !ok {
				continue
			}

			reflect.ValueOf(&t).Elem().SetInt(dval)
		}
	}

	return t, err
}

func (v *InVariable[T]) getIntPtr(val interface{}) (t T, err error) {
	switch v := val.(type) {
	case int64:
		reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&v))
	case float64:
		var d int64
		d = int64(v)
		reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&d))
	case string:
		if v == "" {
			return
		}
		var d int64
		d, err = strconv.ParseInt(v, 10, 64)
		if err == nil {
			reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&d))
		}
	case map[string]interface{}:
		for _, val := range v {
			dval, ok := val.(int64)
			if !ok {
				continue
			}

			reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&dval))
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
	case map[string]interface{}:
		for _, val := range v {
			fval, ok := val.(float64)
			if !ok {
				continue
			}

			reflect.ValueOf(&t).Elem().SetFloat(fval)
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
	case bool:
		reflect.ValueOf(&t).Elem().SetBool(v)
	case string:
		var d bool
		d, err = strconv.ParseBool(v)
		if err == nil {
			reflect.ValueOf(&t).Elem().SetBool(d)
		}
	case map[string]interface{}:
		for _, val := range v {
			bval, ok := val.(bool)
			if !ok {
				continue
			}

			reflect.ValueOf(&t).Elem().SetBool(bval)
		}
	}

	return t, err
}

func (v *InVariable[T]) getBoolPtr(val interface{}) (t T, err error) {
	switch v := val.(type) {
	case int64:
		reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(v > 0))
	case float64:
		reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(v > 0))
	case bool:
		reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&v))
	case string:
		if v == "" {
			return
		}
		var d bool
		d, err = strconv.ParseBool(v)
		if err == nil {
			reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&d))
		}
	case map[string]interface{}:
		for _, val := range v {
			bval, ok := val.(bool)
			if !ok {
				continue
			}

			reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&bval))
		}
	}
	return t, err
}

func (v *InVariable[T]) getString(val interface{}) (t T, err error) {
	switch v := val.(type) {
	case string:
		reflect.ValueOf(&t).Elem().SetString(v)
	case map[string]interface{}:
		for _, val := range v {
			sval, ok := val.(string)
			if !ok {
				continue
			}

			reflect.ValueOf(&t).Elem().SetString(sval)
		}
	}

	return t, err
}

func (v *InVariable[T]) getStringPtr(val interface{}) (t T, err error) {
	switch v := val.(type) {
	case string:
		reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&v))
	case map[string]interface{}:
		for _, val := range v {
			sval, ok := val.(string)
			if !ok {
				continue
			}

			reflect.ValueOf(&t).Elem().Set(reflect.ValueOf(&sval))
		}
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
		val = ctx.Get(v.Name.(string))
		if val == nil {
			return t, nil
		}
		val = deserialize(val)

	}
	kind := reflect.Invalid
	typ := reflect.TypeOf(t)
	if typ != nil {
		kind = typ.Kind()
	}

	if val != nil {
		switch kind {
		case reflect.Ptr:
			switch typ.Elem().Kind() {
			case reflect.Bool:
				return v.getBoolPtr(val)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return v.getIntPtr(val)
			case reflect.String:
				return v.getStringPtr(val)
			}
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

	payload, err := ctx.GetRaw()
	if err != nil {
		return t, err
	}

	val, err = client.GetVariable(&variable{Scope: v.Scope, Name: v.Name.(string), Payload: payload})
	if err != nil {
		return t, err
	}
	val = deserialize(val)
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

		if IsCapnpCapable() {
			info, _ := GetRobotInfo()
			serializedValue, err := robocapnp.Serialize(value, info, v.Name.(string))
			if err != nil {
				return err
			}
			return ctx.Set(v.Name.(string), serializedValue)
		}
		return ctx.Set(v.Name.(string), value)
	}

	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}
	if IsCapnpCapable() {
		info, _ := GetRobotInfo()
		temp, err := robocapnp.Serialize(value, info, v.Name.(string))
		if err != nil {
			return err
		}
		value := temp.(map[string]interface{})
		return client.SetVariable(&variable{Scope: v.Scope, Name: v.Name.(string)}, value)
	} else {
		return client.SetVariable(&variable{Scope: v.Scope, Name: v.Name.(string)}, value)
	}

}

func deserialize(val interface{}) interface{} {
	if IsCapnpCapable() {
		if res, ok := val.(map[string]interface{}); ok {
			if capnp_id, ok := res[robocapnp.ROBOMOTION_CAPNP_ID].(string); ok {
				if strings.HasPrefix(capnp_id, robocapnp.ROBOMOTION_CAPNP_PREFIX) {
					val, _ = robocapnp.Deserialize(capnp_id)

				}
			}

		}
	}
	return val
}
