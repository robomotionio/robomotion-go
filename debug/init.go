package debug

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/mosteknoloji/robomotion-go-lib/proto"
	"bitbucket.org/mosteknoloji/robomotion-go-lib/utils"

	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
)

type Protocol string

const (
	ProtocolInvalid Protocol = ""
	ProtocolNetRPC  Protocol = "netrpc"
	ProtocolGRPC    Protocol = "grpc"
)

type AttachConfig struct {
	Protocol Protocol `json:"protocol"`
	Addr     net.Addr `json:"addr"`
	PID      int      `json:"pid"`
}

const (
	timeout = 30 * time.Second
)

func Attach(listener net.Listener) {

	t1 := time.Now()
	for listener == nil {
		if time.Now().Sub(t1) >= timeout {
			log.Fatalln("timeout: plugin listener is nil")
		}

		time.Sleep(time.Second)
	}

	p := strings.Split(listener.Addr().String(), ":")
	port, err := strconv.Atoi(p[1])
	if err != nil {
		log.Fatalln(err)
	}

	cfg := &AttachConfig{
		Protocol: ProtocolGRPC,
		PID:      os.Getpid(),
		Addr: &net.TCPAddr{
			IP:   net.ParseIP(p[0]),
			Port: port,
		},
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

	runnerCli := proto.NewDebugClient(conn)
	_, err = runnerCli.Attach(context.Background(), &proto.AttachRequest{Config: cfgData})
	if err != nil {
		log.Fatalln(err)
	}
}

func getRPCAddr() string {
	dir := utils.TempDir()
	fileName := "runcfg.json"

	cfgFile := path.Join(dir, fileName)
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		log.Fatalln(err)
	}

	return gjson.Get(string(data), "listen").String()
}
