package message

import (
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type Context struct {
	ID  string
	msg []byte
}

func NewContext(msg []byte) Context {
	return Context{
		ID:  gjson.GetBytes(msg, "id").String(),
		msg: msg,
	}
}

func (ctx Context) Set(path string, value interface{}) (err error) {
	ctx.msg, err = sjson.SetBytes(ctx.msg, path, value)
	return
}

func (ctx Context) Get(path string) interface{} {
	return gjson.GetBytes(ctx.msg, path).Value()
}

func (ctx Context) GetString(path string) string {
	return gjson.GetBytes(ctx.msg, path).String()
}

func (ctx Context) GetBool(path string) bool {
	return gjson.GetBytes(ctx.msg, path).Bool()
}

func (ctx Context) GetInt(path string) int64 {
	return gjson.GetBytes(ctx.msg, path).Int()
}

func (ctx Context) GetFloat(path string) float64 {
	return gjson.GetBytes(ctx.msg, path).Float()
}

func (ctx Context) GetRaw() json.RawMessage {
	return json.RawMessage(ctx.msg)
}
