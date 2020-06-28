package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

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

	n := reflect.New(f.Type)
	handler := n.Interface().(MessageHandler)
	err := json.Unmarshal(config, &handler)
	if err != nil {
		return err
	}

	field := n.Elem().FieldByName("Node")
	if !field.IsValid() {
		return fmt.Errorf("Missing node common properties")
	}

	node := field.Interface().(Node)
	AddNodeHandler(node, handler)
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
