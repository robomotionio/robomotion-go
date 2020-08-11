package debug

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
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
	Addr     string   `json:"addr"`
	PID      int      `json:"pid"`
}

const (
	timeout = 30 * time.Second
)

func Attach() {

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	gAddr := ""
	t1 := time.Now()

	for gAddr == "" {
		if time.Now().Sub(t1) >= timeout {
			log.Fatalln("timeout: plugin listener is nil")
		}

		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			log.Fatal(err)
		}

		line := string(buf[:n])
		if strings.Contains(line, "|") {
			gAddr = strings.Split(line, "|")[3]
		}

		time.Sleep(time.Second)
	}

	os.Stdout = old

	cfg := &AttachConfig{
		Protocol: ProtocolGRPC,
		PID:      os.Getpid(),
		Addr:     gAddr,
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
