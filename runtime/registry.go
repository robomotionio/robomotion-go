package runtime

import (
	"os"
	"os/signal"
	"reflect"
	"time"
	"unsafe"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"

	"bitbucket.org/mosteknoloji/robomotion-go-lib/tl"
)

type PluginNode struct {
	Node
}

var (
	nc   int32
	done = make(chan bool, 1)
)

func Start() {

	if len(os.Args) > 3 && os.Args[1] == "-s" { // generate spec file
		generateSpecFile(os.Args[2], os.Args[3])
		return
	}

	go plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"plugin": &NodePlugin{Impl: &Node{}},
		},
		Logger: hclog.New(&hclog.LoggerOptions{
			Output:     hclog.DefaultOutput,
			Level:      hclog.Trace,
			JSONFormat: true,
		}),
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})

	RegisterFactories()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	signal.Notify(sc, os.Kill)

	go func() {
		<-sc
		done <- true
	}()

	<-done
	time.Sleep(time.Second * 2)
}

func RegisterFactories() {

	types := GetNodeTypes()
	for _, t := range types {
		snode, _ := t.FieldByName("Node")
		spec := snode.Tag.Get("spec")
		nsMap := parseSpec(spec)
		name := nsMap["name"]
		RegisterNodeFactory(name, &NodeFactory{Type: t})
	}
}

func GetNodeTypes() []reflect.Type {

	types := []reflect.Type{}
	sections, offsets := tl.Typelinks()
	for i, base := range sections {
		for _, offset := range offsets[i] {
			typeAddr := tl.Add(base, uintptr(offset), "")
			typ := reflect.TypeOf(*(*interface{})(unsafe.Pointer(&typeAddr)))
			var handler *MessageHandler
			if typ.Implements(reflect.TypeOf(handler).Elem()) {
				types = append(types, typ.Elem())
			}
		}
	}

	return types
}
