package debug

import (
	"context"
	"log"
	"os"

	"github.com/robomotionio/robomotion-go/proto"
	"google.golang.org/grpc"
)

func Detach(namespace string) {
	addr := os.Getenv("ATTACH_TO")
	if addr == "" {
		log.Fatalln("runner RPC address is nil")
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	runnerCli := proto.NewDebugClient(conn)
	_, err = runnerCli.Detach(context.Background(), &proto.DetachRequest{Namespace: namespace})
	if err != nil {
		log.Fatalln(err)
	}
}
