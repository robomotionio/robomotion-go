package runtime

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"reflect"
	"time"
	"unsafe"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/tidwall/gjson"

	"github.com/robomotionio/robomotion-go/debug"
	"github.com/robomotionio/robomotion-go/tl"
)

type PluginNode struct {
	Node
}

var (
	nc       int32
	done     = make(chan bool, 1)
	ns       = ""
	attached = false
	serveCfg = &plugin.ServeConfig{
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
	}
)

func Start() {

	if len(os.Args) > 1 { // start with arg
		arg := os.Args[1]
		config := ReadConfigFile()

		ns := config.Get("namespace").String()
		version := config.Get("version").String()

		switch arg {
		case "-a": // attach
			attached = true
			go debug.Attach(ns)

		case "-s": // generate spec file
			generateSpecFile(ns, version)
			return
		}
	}

	go plugin.Serve(serveCfg)

	RegisterFactories()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	signal.Notify(sc, os.Kill)

	go func() {
		<-sc
		done <- true
	}()

	<-done

	if attached {
		debug.Detach(ns)
	}

	time.Sleep(time.Second * 2)
}

func RegisterFactories() {

	types := GetNodeTypes()
	for _, t := range types {
		snode, _ := t.FieldByName("Node")
		spec := snode.Tag.Get("spec")
		nsMap := parseSpec(spec)
		name := nsMap["id"]
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

func ReadConfigFile() gjson.Result {
	d, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Read config error: %+v", err)
	}

	return gjson.ParseBytes(d)
}
