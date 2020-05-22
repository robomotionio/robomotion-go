package runtime

import (
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var (
	RunnerConn *grpc.ClientConn
)

func CheckRunnerConn() {
	for {
		state := RunnerConn.GetState()

		switch state {
		case connectivity.Connecting, connectivity.Idle, connectivity.Ready:
			break
		default:
			os.Exit(1)
		}

		time.Sleep(1 * time.Second)
	}
}
