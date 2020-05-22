package runtime

import (
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
			done <- true
		}

		time.Sleep(1 * time.Second)
	}
}
