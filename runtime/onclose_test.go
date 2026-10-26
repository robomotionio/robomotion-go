package runtime

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/robomotionio/robomotion-go/message"
	"github.com/robomotionio/robomotion-go/proto"
)

type closeBarrierNode struct{ release <-chan struct{} }

func (n *closeBarrierNode) OnCreate() error                     { return nil }
func (n *closeBarrierNode) OnMessage(ctx message.Context) error { return nil }
func (n *closeBarrierNode) OnClose() error {
	<-n.release
	return nil
}

// The robot closes a package's nodes concurrently. Only one OnClose may
// release the LMO store: closing its zstd decoder twice panics with
// "close of closed channel" and takes the package down mid-close.
func TestOnCloseConcurrentReleasesStoreOnce(t *testing.T) {
	const nodes = 16
	server := &GRPCServer{}

	for iter := 0; iter < 200; iter++ {
		if err := InitLMOStore(); err != nil {
			t.Fatal(err)
		}
		release := make(chan struct{})
		for i := 0; i < nodes; i++ {
			AddNodeHandler(Node{GUID: fmt.Sprintf("n%d", i)}, &closeBarrierNode{release: release})
		}
		nc = nodes

		var wg sync.WaitGroup
		for i := 0; i < nodes; i++ {
			wg.Add(1)
			go func(guid string) {
				defer wg.Done()
				if _, err := server.OnClose(context.Background(), &proto.OnCloseRequest{Guid: guid}); err != nil {
					t.Error(err)
				}
			}(fmt.Sprintf("n%d", i))
		}
		close(release)
		closed := make(chan struct{})
		go func() { wg.Wait(); close(closed) }()
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			// A second `done <- true` blocks once the buffer is full.
			t.Fatalf("iteration %d: OnClose calls did not return", iter)
		}

		select {
		case <-done:
		default:
			t.Fatalf("iteration %d: no done signal after the last node closed", iter)
		}
		select {
		case <-done:
			t.Fatalf("iteration %d: done signalled more than once", iter)
		default:
		}
		if lmoStore != nil {
			t.Fatalf("iteration %d: LMO store still open", iter)
		}
	}
}
