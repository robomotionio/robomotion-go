package message

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type Context interface {
	GetID() string
	Set(path string, value interface{}) error
	Get(path string) interface{}
	GetString(path string) string
	GetBool(path string) bool
	GetInt(path string) int64
	GetFloat(path string) float64
	GetRaw() json.RawMessage
	SetRaw(data json.RawMessage)
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

func (msg *message) Set(path string, value interface{}) (err error) {
	path = strings.ReplaceAll(path, "[", ".")
	path = strings.ReplaceAll(path, "]", "")
	msg.data, err = sjson.SetBytes(msg.data, path, value)
	return
}

func (msg *message) GetID() string {
	return msg.ID
}

func (msg *message) Get(path string) interface{} {
	return gjson.GetBytes(msg.data, path).Value()
}

func (msg *message) GetString(path string) string {
	return gjson.GetBytes(msg.data, path).String()
}

func (msg *message) GetBool(path string) bool {
	return gjson.GetBytes(msg.data, path).Bool()
}

func (msg *message) GetInt(path string) int64 {
	return gjson.GetBytes(msg.data, path).Int()
}

func (msg *message) GetFloat(path string) float64 {
	return gjson.GetBytes(msg.data, path).Float()
}

func (msg *message) GetRaw() json.RawMessage {
	return json.RawMessage(msg.data)
}

func (msg *message) SetRaw(data json.RawMessage) {
	msg.data = data
}

func (msg *message) IsEmpty() bool {
	return msg.data == nil || len(msg.data) == 0
}
