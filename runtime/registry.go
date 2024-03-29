package runtime

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"reflect"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/robomotionio/go-plugin"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/robomotionio/robomotion-go/debug"
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

func initLogger() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetOutput(os.Stderr)
}

func Start() {

	var config gjson.Result
	if len(os.Args) > 1 { // start with arg
		arg := os.Args[1]
		config = ReadConfigFile()

		name := config.Get("name").String()
		version := config.Get("version").String()

		switch arg {
		case "-a": // attach
			attached = true

		case "-s": // generate spec file
			generateSpecFile(name, version)
			return
		}
	}

	initLogger()
	os.Setenv(serveCfg.MagicCookieKey, serveCfg.MagicCookieValue)

	go plugin.Serve(serveCfg)
	if attached {
		ns = config.Get("namespace").String()
		go debug.Attach(ns, serveCfg)
	}

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
	nodes := RegisteredNodes()
	for _, node := range nodes {
		typ := reflect.TypeOf(node)
		types = append(types, typ.Elem())
	}

	return types
}

func ReadConfigFile() gjson.Result {
	d, err := ioutil.ReadFile("config.json")
	if err == nil {
		return gjson.ParseBytes(d)
	}

	d, err = ioutil.ReadFile("../config.json")
	if err != nil {
		log.Fatalf("Read config error: %+v", err)
	}

	return gjson.ParseBytes(d)
}
