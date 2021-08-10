package debug

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	plugin "github.com/mosteknoloji/go-plugin"
	"github.com/robomotionio/robomotion-go/proto"

	"github.com/cakturk/go-netstat/netstat"
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

	addr := getRPCAddr()
	if addr == "" {
		log.Fatalln("runner RPC address is nil")
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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

func getRPCAddr() string {
	tabs, err := netstat.TCPSocks(func(s *netstat.SockTabEntry) bool {
		return s.State == netstat.Listen && s.Process != nil && strings.Contains(s.Process.Name, "robomotion-runner")
	})
	if err != nil {
		log.Println(err)
		return ""
	}
	if len(tabs) == 0 {
		return ""
	}
	return tabs[len(tabs)-1].LocalAddr.String()
}
