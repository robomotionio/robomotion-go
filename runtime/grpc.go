package runtime

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	st "github.com/golang/protobuf/ptypes/struct"
	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/mosteknoloji/go-plugin"
	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/robomotionio/robomotion-go/message"
	"github.com/robomotionio/robomotion-go/proto"
)

var (
	conn   *grpc.ClientConn
	client RuntimeHelper
)

type GRPCServer struct {
	broker *plugin.GRPCBroker

	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl INode
}

func (m *GRPCServer) Init(ctx context.Context, req *proto.InitRequest) (*proto.Empty, error) {

	var err error
	conn, err = m.broker.Dial(req.EventServer)
	if err != nil {
		hclog.Default().Info("grpc.server.init", "err", err)
		return nil, err
	}

	go checkConnState()
	e := &GRPCRuntimeHelperClient{proto.NewRuntimeHelperClient(conn)}

	m.Impl.Init(e)
	return &proto.Empty{}, err
}

func (m *GRPCServer) OnCreate(ctx context.Context, req *proto.OnCreateRequest) (*proto.OnCreateResponse, error) {

	resp := &proto.OnCreateResponse{}

	f := GetNodeFactory(req.Name)
	if f == nil {
		return nil, fmt.Errorf("%s factory not found", req.Name)
	}

	err := f.OnCreate(context.TODO(), req.Config)
	if err != nil {
		hclog.Default().Info("grpc.server.oncreate.factory", "err", err)
		return resp, err
	}

	guid := gjson.Get(string(req.Config), "guid").String()

	node := GetNodeHandler(guid)
	if node == nil {
		hclog.Default().Info("grpc.server.oncreate.node", "err", "no handler")
	}

	err = node.Handler.OnCreate()
	if err != nil {
		hclog.Default().Info("grpc.server.oncreate.node", "err", err)
		return resp, err
	}

	atomic.AddInt32(&nc, 1)

	return resp, err
}

func (m *GRPCServer) OnMessage(ctx context.Context, req *proto.OnMessageRequest) (*proto.OnMessageResponse, error) {

	resp := &proto.OnMessageResponse{}
	data, err := Decompress(req.InMessage)
	if err != nil {
		hclog.Default().Info("grpc.server.onmessage", "err", err)
		return resp, err
	}

	node := GetNodeHandler(req.Guid)
	if node == nil {
		hclog.Default().Info("grpc.server.oncreate.node", "err", "no handler")
	}

	msgCtx := message.NewContext(data)

	time.Sleep(time.Duration(node.DelayBefore*1000) * time.Millisecond)
	err = node.Handler.OnMessage(msgCtx)
	if err != nil && node.ContinueOnError {
		err = nil
	}
	resp.OutMessage = []byte(msgCtx.GetRaw())
	time.Sleep(time.Duration(node.DelayAfter*1000) * time.Millisecond)

	return resp, err
}

func (m *GRPCServer) OnClose(ctx context.Context, req *proto.OnCloseRequest) (*proto.OnCloseResponse, error) {

	node := GetNodeHandler(req.Guid)
	if node == nil {
		return nil, fmt.Errorf("No handler")
	}

	err := node.Handler.OnClose()
	atomic.AddInt32(&nc, -1)
	defer func() {
		if atomic.LoadInt32(&nc) == 0 {
			defer func() {
				done <- true
			}()
		}
	}()
	return &proto.OnCloseResponse{}, err
}

// GRPCClient is an implementation of KV that talks over RPC.
type GRPCRuntimeHelperClient struct{ client proto.RuntimeHelperClient }

func (m *GRPCRuntimeHelperClient) Close() error {

	_, err := m.client.Close(context.Background(), &proto.Empty{})
	if err != nil {
		hclog.Default().Info("runtime.close", "err", err)
		return err
	}

	return nil
}

func (m *GRPCRuntimeHelperClient) Debug(guid, name string, message interface{}) error {

	msgData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = m.client.Debug(context.Background(), &proto.DebugRequest{
		Guid:    guid,
		Name:    name,
		Message: msgData,
	})

	if err != nil {
		hclog.Default().Info("runtime.debug", "err", err)
		return err
	}

	return nil
}

func (m *GRPCRuntimeHelperClient) EmitFlowEvent(guid, name string) error {

	_, err := m.client.EmitFlowEvent(context.Background(), &proto.EmitFlowEventRequest{
		Guid: guid,
		Name: name,
	})

	if err != nil {
		hclog.Default().Info("runtime.flow", "err", err)
		return err
	}

	return nil
}

func (m *GRPCRuntimeHelperClient) EmitInput(guid string, input []byte) error {

	_, err := m.client.EmitInput(context.Background(), &proto.EmitInputRequest{
		Guid:  guid,
		Input: input,
	})

	if err != nil {
		hclog.Default().Info("runtime.input", "err", err)
		return err
	}

	return nil
}

func (m *GRPCRuntimeHelperClient) EmitOutput(guid string, output []byte, port int32) error {

	_, err := m.client.EmitOutput(context.Background(), &proto.EmitOutputRequest{
		Guid:   guid,
		Output: output,
		Port:   port,
	})

	if err != nil {
		hclog.Default().Info("runtime.output", "err", err)
		return err
	}

	return nil
}

func (m *GRPCRuntimeHelperClient) EmitError(guid, name, message string) error {

	_, err := m.client.EmitError(context.Background(), &proto.EmitErrorRequest{
		Guid:    guid,
		Name:    name,
		Message: message,
	})

	if err != nil {
		hclog.Default().Info("runtime.error", "err", err)
		return err
	}

	return nil
}

func (m *GRPCRuntimeHelperClient) GetVaultItem(vaultID, itemID string) (map[string]interface{}, error) {

	resp, err := m.client.GetVaultItem(context.Background(), &proto.GetVaultItemRequest{
		ItemId:  itemID,
		VaultId: vaultID,
	})

	if err != nil {
		hclog.Default().Info("runtime.getvaultitem", "err", err)
		return nil, err
	}

	return parseStruct(resp.Item).(map[string]interface{}), nil
}

func (m *GRPCRuntimeHelperClient) GetVariable(variable *Variable) (interface{}, error) {

	v := &proto.Variable{
		Name:  variable.Name,
		Scope: variable.Scope,
	}

	resp, err := m.client.GetVariable(context.Background(), &proto.GetVariableRequest{
		Variable: v,
	})

	if err != nil {
		hclog.Default().Info("runtime.getstringvariable", "err", err)
		return "", err
	}

	return parseStruct(resp.Value), nil
}

func (m *GRPCRuntimeHelperClient) SetVariable(variable *Variable, value interface{}) error {

	v := &proto.Variable{
		Name:  variable.Name,
		Scope: variable.Scope,
	}

	fields := make(map[string]*st.Value)
	fields["value"] = ToValue(value)
	val := &st.Struct{Fields: fields}

	_, err := m.client.SetVariable(context.Background(), &proto.SetVariableRequest{
		Variable: v,
		Value:    val,
	})

	if err != nil {
		hclog.Default().Info("runtime.setstringvariable", "err", err)
		return err
	}

	return nil
}

func checkConnState() {

	for {
		state := conn.GetState()

		switch state {
		case connectivity.Connecting, connectivity.Idle, connectivity.Ready:
			break
		default:
			done <- true
		}

		time.Sleep(1 * time.Second)
	}
}
