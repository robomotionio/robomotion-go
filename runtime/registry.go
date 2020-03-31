package runtime

import (
	"os"
	"reflect"
	"sync"
	"unsafe"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"bitbucket.org/mosteknoloji/robomotion-go-lib/tl"
)

type PluginNode struct {
	SNode
}

var (
	wg      sync.WaitGroup
	started = false
)

func Start() {

	if len(os.Args) > 1 && os.Args[1] == "-s" { // generate spec file
		return
	}

	go plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"plugin": &NodePlugin{Impl: &SNode{}},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})

	Init()

	wg.Add(1)
	wg.Wait()

	err := Close()
	if err != nil {
		hclog.Default().Info("plugin.close", "err", err)
		os.Exit(0)
	}

}

func Init() {

	types := GetNodeTypes()
	for _, t := range types {
		snode, _ := t.FieldByName("SNode")
		name := snode.Tag.Get("name")
		CreateNode(name, &NodeFactory{Type: t})
	}

	hclog.Default().Info("nodes", "map", Factories())
}

func GetNodeTypes() []reflect.Type {

	types := []reflect.Type{}
	sections, offsets := tl.Typelinks()
	for i, base := range sections {
		for _, offset := range offsets[i] {
			typeAddr := tl.Add(base, uintptr(offset), "")
			typ := reflect.TypeOf(*(*interface{})(unsafe.Pointer(&typeAddr)))

			var node *Node
			if typ.Implements(reflect.TypeOf(node).Elem()) {
				types = append(types, typ.Elem())
			}
		}
	}

	return types
}

func WaiterAdd() {

	wg.Add(1)
	if !started {
		started = true
		wg.Done()
	}
}

func WaiterDone() {
	wg.Done()
}
