package debug

import (
	"context"
	"log"

	"github.com/robomotionio/robomotion-go/proto"
	"google.golang.org/grpc"
)

func Detach(namespace string) {
	if attachedTo == "" {
		log.Fatalln("empty gRPC address")
	}

	conn, err := grpc.Dial(attachedTo, grpc.WithInsecure())
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
