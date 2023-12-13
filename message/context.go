package message

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type GetOption func(json.RawMessage) (json.RawMessage, error)
type SetOption func(json.RawMessage) (json.RawMessage, error)

var (
	GetRaw func(json.RawMessage, ...GetOption) (json.RawMessage, error)
	SetRaw func(json.RawMessage, ...SetOption) (json.RawMessage, error)
)

type Context interface {
	GetID() string
	Set(path string, value interface{}) error
	Get(path string) interface{}
	GetString(path string) string
	GetBool(path string) bool
	GetInt(path string) int64
	GetFloat(path string) float64
	GetRaw(options ...GetOption) (json.RawMessage, error)
	SetRaw(data json.RawMessage, options ...SetOption) error
	IsEmpty() bool
}

type message struct {
	ID   string
	data []byte
}

func NewContext(data []byte) Context {
	return &message{
		ID:   gjson.GetBytes(data, "id").String(),
		data: data,
	}
}

func convertPath(path string) string {
	path = strings.ReplaceAll(path, "[", ".")
	path = strings.ReplaceAll(path, "]", "")
	return path
}

func (msg *message) Set(path string, value interface{}) (err error) {
	path = convertPath(path)
	msg.data, err = sjson.SetBytes(msg.data, path, value)
	return
}

func (msg *message) GetID() string {
	return msg.ID
}

func (msg *message) get(path string) gjson.Result {
	path = convertPath(path)
	return gjson.GetBytes(msg.data, path)
}

func (msg *message) Get(path string) interface{} {
	return msg.get(path).Value()
}

func (msg *message) GetString(path string) string {
	return msg.get(path).String()
}

func (msg *message) GetBool(path string) bool {
	return msg.get(path).Bool()
}

func (msg *message) GetInt(path string) int64 {
	return msg.get(path).Int()
}

func (msg *message) GetFloat(path string) float64 {
	return msg.get(path).Float()
}

func (msg *message) GetRaw(options ...GetOption) (json.RawMessage, error) {
	return GetRaw(msg.data, options...)
}

func (msg *message) SetRaw(data json.RawMessage, options ...SetOption) (err error) {
	msg.data, err = SetRaw(data, options...)
	return err
}

func (msg *message) IsEmpty() bool {
	return msg.data == nil || len(msg.data) == 0
}
