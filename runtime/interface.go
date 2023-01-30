// Package shared contains shared data between the host and plugins.
package runtime

import (
	plugin "github.com/robomotionio/go-plugin"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/robomotionio/robomotion-go/proto"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "robomotion_plugin",
	MagicCookieValue: "6e80b1a2cf26c5935ed7b6e5be77fe218d5f358d",
}

type RuntimeHelper interface {
	Close() error
	Debug(string, string, interface{}) error
	EmitFlowEvent(string, string) error
	EmitInput(string, []byte) error
	EmitOutput(string, []byte, int32) error
	EmitError(string, string, string) error
	GetVaultItem(string, string) (map[string]interface{}, error)
	SetVaultItem(string, string, []byte) (map[string]interface{}, error)
	GetVariable(*variable) (interface{}, error)
	SetVariable(*variable, interface{}) error
	GetRobotInfo() (map[string]interface{}, error)
	AppRequest([]byte) ([]byte, error)
}

// This is the implementation of plugin.Plugin so we can serve/consume this.
// We also implement GRPCPlugin so that this plugin can be served over
// gRPC.
type NodePlugin struct {
	plugin.NetRPCUnsupportedPlugin

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
