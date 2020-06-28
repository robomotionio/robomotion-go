package runtime

import (
	"encoding/json"
	"reflect"

	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
)

type INodeFactory interface {
	OnCreate(ctx context.Context, config []byte) error
}

type NodeFactory struct {
	Type reflect.Type
}

func (f *NodeFactory) OnCreate(ctx context.Context, config []byte) error {

	node := reflect.New(f.Type).Interface().(Node)
	err := json.Unmarshal(config, &node)
	if err != nil {
		return err
	}

	guid := gjson.GetBytes(config, "guid").String()
	AddNode(guid, node)
	return nil
}
