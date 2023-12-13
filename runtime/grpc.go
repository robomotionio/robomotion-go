package runtime

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	st "github.com/golang/protobuf/ptypes/struct"
	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/robomotionio/go-plugin"
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
	proto.UnimplementedNodeServer
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

	resp := &proto.OnMessageResponse{OutMessage: nil}
	data, err := Decompress(req.InMessage)
	if err != nil {
		hclog.Default().Info("grpc.server.onmessage", "err", err)
		return resp, err
	}

	node := GetNodeHandler(req.Guid)
	if node == nil {
		hclog.Default().Info("grpc.server.oncreate.node", "err", "no handler")
		return nil, fmt.Errorf("node handler not found")
	}

	msgCtx := message.NewContext(data)

	time.Sleep(time.Duration(node.DelayBefore*1000) * time.Millisecond)
	err = node.Handler.OnMessage(msgCtx)
	if err != nil && node.ContinueOnError {
		err = nil
	}

	if !msgCtx.IsEmpty() {
		resp.OutMessage, _ = msgCtx.GetRaw()
	}

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

func (m *GRPCServer) GetCapabilities(ctx context.Context, req *proto.Empty) (*proto.PGetCapabilitiesResponse, error) {
	return &proto.PGetCapabilitiesResponse{Capabilities: uint64(packageCapabilities)}, nil
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

func (m *GRPCRuntimeHelperClient) SetVaultItem(vaultID, itemID string, data []byte) (map[string]interface{}, error) {
	resp, err := m.client.SetVaultItem(context.Background(), &proto.SetVaultItemRequest{
		ItemId:  itemID,
		VaultId: vaultID,
		Data:    data,
	})

	if err != nil {
		hclog.Default().Info("runtime.setvaultitem", "err", err)
		return nil, err
	}

	return parseStruct(resp.Item).(map[string]interface{}), nil
}

func (m *GRPCRuntimeHelperClient) GetVariable(variable *variable) (interface{}, error) {

	v := &proto.Variable{
		Name:    variable.Name,
		Scope:   variable.Scope,
		Payload: variable.Payload,
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

func (m *GRPCRuntimeHelperClient) SetVariable(variable *variable, value interface{}) error {

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

func (m *GRPCRuntimeHelperClient) AppRequest(request []byte, timeout int32) ([]byte, error) {

	resp, err := m.client.AppRequest(context.Background(), &proto.AppRequestRequest{
		Request: request,
		Timeout: timeout,
	})

	if err != nil {
		hclog.Default().Info("runtime.apprequest", "err", err)
		return nil, err
	}

	return resp.Response, nil
}

func (m *GRPCRuntimeHelperClient) AppRequestV2(request []byte) ([]byte, error) {

	resp, err := m.client.AppRequestV2(context.Background(), &proto.AppRequestV2Request{
		Request: request,
	})

	if err != nil {
		hclog.Default().Info("runtime.apprequest", "err", err)
		return nil, err
	}

	return resp.Response, nil
}

func (m *GRPCRuntimeHelperClient) AppPublish(request []byte) error {

	_, err := m.client.AppPublish(context.Background(), &proto.AppPublishRequest{
		Request: request,
	})

	if err != nil {
		hclog.Default().Info("runtime.apppublish", "err", err)
		return err
	}

	return nil
}

func (m *GRPCRuntimeHelperClient) AppDownload(id, dir, file string) (string, error) {

	resp, err := m.client.AppDownload(context.Background(), &proto.AppDownloadRequest{
		Directory: dir,
		File:      file,
		Id:        id,
	})

	if err != nil {
		hclog.Default().Info("runtime.appdownload", "err", err)
		return "", err
	}

	return resp.Path, nil
}

func (m *GRPCRuntimeHelperClient) AppUpload(id, path string) (string, error) {

	resp, err := m.client.AppUpload(context.Background(), &proto.AppUploadRequest{
		Id:   id,
		Path: path,
	})

	if err != nil {
		hclog.Default().Info("runtime.appupload", "err", err)
		return "", err
	}

	return resp.Url, nil
}

func (m *GRPCRuntimeHelperClient) GatewayRequest(method, endpoint, body string, headers map[string]string) (*proto.GatewayRequestResponse, error) {

	resp, err := m.client.GatewayRequest(context.Background(), &proto.GatewayRequestRequest{
		Method:   method,
		Endpoint: endpoint,
		Body:     body,
		Headers:  headers,
	})

	if err != nil {
		hclog.Default().Info("runtime.appupload", "err", err)
		return nil, err
	}

	return resp, nil
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

func (m *GRPCRuntimeHelperClient) GetRobotInfo() (map[string]interface{}, error) {
	resp, err := m.client.GetRobotInfo(context.Background(), &proto.Empty{})

	if err != nil {
		hclog.Default().Info("runtime.getrobotinfo", "err", err)
		return nil, err
	}

	return parseStruct(resp.Robot).(map[string]interface{}), nil
}
