package debug

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	plugin "github.com/robomotionio/go-plugin"
	"github.com/robomotionio/robomotion-go/proto"

	"google.golang.org/grpc"
)

type Protocol string

const (
	ProtocolInvalid Protocol = ""
	ProtocolNetRPC  Protocol = "netrpc"
	ProtocolGRPC    Protocol = "grpc"
)

type AttachConfig struct {
	Protocol  Protocol `json:"protocol"`
	Addr      string   `json:"addr"`
	PID       int      `json:"pid"`
	Namespace string   `json:"namespace"`
}

const (
	timeout = 30 * time.Second
)

var (
	attachedTo string
)

func Attach(namespace string, opts *plugin.ServeConfig) {

	gAddr := ""
	t1 := time.Now()

	for gAddr == "" {
		if time.Now().Sub(t1) >= timeout {
			log.Fatalln("timeout: plugin listener is nil")
		}

		if opts.Listener != nil {
			gAddr = opts.Listener.Addr().String()
		}

		time.Sleep(time.Second)
	}

	cfg := &AttachConfig{
		Protocol:  ProtocolGRPC,
		PID:       os.Getpid(),
		Addr:      gAddr,
		Namespace: namespace,
	}

	cfgData, err := json.Marshal(cfg)
	if err != nil {
		log.Fatalln(err)
	}

	attachedTo = getRPCAddr()
	if attachedTo == "" {
		log.Fatalln("empty gRPC address")
	}

	log.Printf("Attached to %s", attachedTo)

	conn, err := grpc.Dial(attachedTo, grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	runnerCli := proto.NewDebugClient(conn)
	_, err = runnerCli.Attach(context.Background(), &proto.AttachRequest{Config: cfgData})
	if err != nil {
		log.Fatalln(err)
	}
}
