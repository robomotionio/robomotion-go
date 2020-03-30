package runtime

import (
	"encoding/json"

	st "github.com/golang/protobuf/ptypes/struct"
	"github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/tidwall/gjson"
	"golang.org/x/net/context"

	"bitbucket.org/mosteknoloji/robomotion/robomotion-plugin/go-lib/proto"
)

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl INode

	broker *plugin.GRPCBroker
}

func (m *GRPCServer) Init(ctx context.Context, req *proto.InitRequest) (*proto.Empty, error) {

	conn, err := m.broker.Dial(req.EventServer)
	if err != nil {
		hclog.Default().Info("grpc.server.init", "err", err)
		return nil, err
	}
	//defer conn.Close()

	e := &GRPCRuntimeHelperClient{proto.NewRuntimeHelperClient(conn)}

	err = m.Impl.Init(e)
	return &proto.Empty{}, err
}

func (m *GRPCServer) OnCreate(ctx context.Context, req *proto.OnCreateRequest) (*proto.OnCreateResponse, error) {

	resp := &proto.OnCreateResponse{}
	err := Factories()[req.Name].OnCreate(context.TODO(), req.Config)
	if err != nil {
		hclog.Default().Info("grpc.server.oncreate.factory", "err", err)
		return resp, err
	}

	guid := gjson.Get(string(req.Config), "guid").String()
	err = Nodes()[guid].OnCreate(req.Config)
	if err != nil {
		hclog.Default().Info("grpc.server.oncreate.node", "err", err)
		return resp, err
	}

	WaiterAdd()
	return resp, err
}

func (m *GRPCServer) OnMessage(ctx context.Context, req *proto.OnMessageRequest) (*proto.OnMessageResponse, error) {

	resp := &proto.OnMessageResponse{}
	data, err := Decompress(req.InMessage)
	if err != nil {
		hclog.Default().Info("grpc.server.onmessage", "err", err)
		return resp, err
	}

	node := Nodes()[req.Guid]
	resp.OutMessage, err = node.OnMessage(data)
	return resp, err
}

func (m *GRPCServer) OnClose(ctx context.Context, req *proto.OnCloseRequest) (*proto.OnCloseResponse, error) {

	WaiterDone()
	node := Nodes()[req.Guid]
	err := node.OnClose()
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

func (m *GRPCRuntimeHelperClient) GetIntVariable(variable *Variable, message []byte) (int32, error) {

	v := &proto.Variable{
		Name:  variable.Name,
		Scope: variable.Scope,
	}

	resp, err := m.client.GetIntVariable(context.Background(), &proto.GetVariableRequest{
		Variable: v,
		Message:  message,
	})

	if err != nil {
		hclog.Default().Info("runtime.getintvariable", "err", err)
		return 0, err
	}

	return resp.Value, nil
}

func (m *GRPCRuntimeHelperClient) GetStringVariable(variable *Variable, message []byte) (string, error) {

	v := &proto.Variable{
		Name:  variable.Name,
		Scope: variable.Scope,
	}

	resp, err := m.client.GetStringVariable(context.Background(), &proto.GetVariableRequest{
		Variable: v,
		Message:  message,
	})

	if err != nil {
		hclog.Default().Info("runtime.getstringvariable", "err", err)
		return "", err
	}

	return resp.Value, nil
}

func (m *GRPCRuntimeHelperClient) GetInterfaceVariable(variable *Variable, message []byte) (interface{}, error) {

	v := &proto.Variable{
		Name:  variable.Name,
		Scope: variable.Scope,
	}

	resp, err := m.client.GetInterfaceVariable(context.Background(), &proto.GetVariableRequest{
		Variable: v,
		Message:  message,
	})

	if err != nil {
		hclog.Default().Info("runtime.getstringvariable", "err", err)
		return "", err
	}

	return parseStruct(resp.Value), nil
}

func (m *GRPCRuntimeHelperClient) SetIntVariable(variable *Variable, message []byte, value int32) ([]byte, error) {

	v := &proto.Variable{
		Name:  variable.Name,
		Scope: variable.Scope,
	}

	resp, err := m.client.SetIntVariable(context.Background(), &proto.SetIntVariableRequest{
		Variable: v,
		Message:  message,
		Value:    value,
	})

	if err != nil {
		hclog.Default().Info("runtime.setintvariable", "err", err)
		return message, err
	}

	return resp.Message, nil
}

func (m *GRPCRuntimeHelperClient) SetStringVariable(variable *Variable, message []byte, value string) ([]byte, error) {

	v := &proto.Variable{
		Name:  variable.Name,
		Scope: variable.Scope,
	}

	resp, err := m.client.SetStringVariable(context.Background(), &proto.SetStringVariableRequest{
		Variable: v,
		Message:  message,
		Value:    value,
	})

	if err != nil {
		hclog.Default().Info("runtime.setstringvariable", "err", err)
		return message, err
	}

	return resp.Message, nil
}

func (m *GRPCRuntimeHelperClient) SetInterfaceVariable(variable *Variable, message []byte, value interface{}) ([]byte, error) {

	v := &proto.Variable{
		Name:  variable.Name,
		Scope: variable.Scope,
	}

	fields := make(map[string]*st.Value)
	fields["value"] = ToValue(value)
	val := &st.Struct{Fields: fields}

	resp, err := m.client.SetInterfaceVariable(context.Background(), &proto.SetInterfaceVariableRequest{
		Variable: v,
		Message:  message,
		Value:    val,
	})

	if err != nil {
		hclog.Default().Info("runtime.setstringvariable", "err", err)
		return message, err
	}

	return resp.Message, nil
}
