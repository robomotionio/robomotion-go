package runtime

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
)

var (
	factories = make(map[string]INodeFactory)
	fMux      sync.Mutex
)

type INodeFactory interface {
	OnCreate(ctx context.Context, config []byte) error
}

type NodeFactory struct {
	Type reflect.Type
}

func (f *NodeFactory) OnCreate(ctx context.Context, config []byte) error {

	handler := reflect.New(f.Type).Interface().(MessageHandler)
	err := json.Unmarshal(config, &handler)
	if err != nil {
		return err
	}

	guid := gjson.GetBytes(config, "guid").String()
	AddMessageHandler(guid, handler)
	return nil
}

func RegisterNodeFactory(name string, factory INodeFactory) {
	fMux.Lock()
	defer fMux.Unlock()
	factories[name] = factory
}

func GetNodeFactory(name string) INodeFactory {
	fMux.Lock()
	defer fMux.Unlock()
	f, _ := factories[name]
	return f
}
