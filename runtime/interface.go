// Package shared contains shared data between the host and plugins.
package runtime

import (
	"bitbucket.org/mosteknoloji/robomotion-go-lib/proto"
	plugin "github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

type RuntimeHelper interface {
	Close() error
	Debug(string, string, interface{}) error
	EmitFlowEvent(string, string) error
	EmitOutput(string, []byte, int32) error
	EmitError(string, string, string) error
	GetVaultItem(string, string) (map[string]interface{}, error)
	GetIntVariable(*Variable) (int32, error)
	GetStringVariable(*Variable) (string, error)
	GetInterfaceVariable(*Variable) (interface{}, error)
	SetVariable(*Variable, interface{}) error
}

// KV is the interface that we're exposing as a plugin.
type Node interface {
	OnCreate([]byte) error
	OnMessage([]byte) ([]byte, error)
	OnClose() error
}

type INode interface {
	Init(RuntimeHelper) error
}

// This is the implementation of plugin.Plugin so we can serve/consume this.
// We also implement GRPCPlugin so that this plugin can be served over
// gRPC.
type NodePlugin struct {
	plugin.NetRPCUnsupportedPlugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl INode
}

func (p *NodePlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterNodeServer(s, &GRPCServer{
		Impl:   p.Impl,
		broker: broker,
	})
	return nil
}

func (p *NodePlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return nil, nil
}

var _ plugin.GRPCPlugin = &NodePlugin{}
