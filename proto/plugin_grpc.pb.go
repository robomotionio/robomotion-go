// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: plugin.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// NodeClient is the client API for Node service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type NodeClient interface {
	Init(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*Empty, error)
	OnCreate(ctx context.Context, in *OnCreateRequest, opts ...grpc.CallOption) (*OnCreateResponse, error)
	OnMessage(ctx context.Context, in *OnMessageRequest, opts ...grpc.CallOption) (*OnMessageResponse, error)
	OnClose(ctx context.Context, in *OnCloseRequest, opts ...grpc.CallOption) (*OnCloseResponse, error)
}

type nodeClient struct {
	cc grpc.ClientConnInterface
}

func NewNodeClient(cc grpc.ClientConnInterface) NodeClient {
	return &nodeClient{cc}
}

func (c *nodeClient) Init(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/proto.Node/Init", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) OnCreate(ctx context.Context, in *OnCreateRequest, opts ...grpc.CallOption) (*OnCreateResponse, error) {
	out := new(OnCreateResponse)
	err := c.cc.Invoke(ctx, "/proto.Node/OnCreate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) OnMessage(ctx context.Context, in *OnMessageRequest, opts ...grpc.CallOption) (*OnMessageResponse, error) {
	out := new(OnMessageResponse)
	err := c.cc.Invoke(ctx, "/proto.Node/OnMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) OnClose(ctx context.Context, in *OnCloseRequest, opts ...grpc.CallOption) (*OnCloseResponse, error) {
	out := new(OnCloseResponse)
	err := c.cc.Invoke(ctx, "/proto.Node/OnClose", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NodeServer is the server API for Node service.
// All implementations must embed UnimplementedNodeServer
// for forward compatibility
type NodeServer interface {
	Init(context.Context, *InitRequest) (*Empty, error)
	OnCreate(context.Context, *OnCreateRequest) (*OnCreateResponse, error)
	OnMessage(context.Context, *OnMessageRequest) (*OnMessageResponse, error)
	OnClose(context.Context, *OnCloseRequest) (*OnCloseResponse, error)
	mustEmbedUnimplementedNodeServer()
}

// UnimplementedNodeServer must be embedded to have forward compatible implementations.
type UnimplementedNodeServer struct {
}

func (UnimplementedNodeServer) Init(context.Context, *InitRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Init not implemented")
}
func (UnimplementedNodeServer) OnCreate(context.Context, *OnCreateRequest) (*OnCreateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnCreate not implemented")
}
func (UnimplementedNodeServer) OnMessage(context.Context, *OnMessageRequest) (*OnMessageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnMessage not implemented")
}
func (UnimplementedNodeServer) OnClose(context.Context, *OnCloseRequest) (*OnCloseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnClose not implemented")
}
func (UnimplementedNodeServer) mustEmbedUnimplementedNodeServer() {}

// UnsafeNodeServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to NodeServer will
// result in compilation errors.
type UnsafeNodeServer interface {
	mustEmbedUnimplementedNodeServer()
}

func RegisterNodeServer(s grpc.ServiceRegistrar, srv NodeServer) {
	s.RegisterService(&Node_ServiceDesc, srv)
}

func _Node_Init_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).Init(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Node/Init",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).Init(ctx, req.(*InitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_OnCreate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OnCreateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).OnCreate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Node/OnCreate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).OnCreate(ctx, req.(*OnCreateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_OnMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OnMessageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).OnMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Node/OnMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).OnMessage(ctx, req.(*OnMessageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_OnClose_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OnCloseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).OnClose(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Node/OnClose",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).OnClose(ctx, req.(*OnCloseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Node_ServiceDesc is the grpc.ServiceDesc for Node service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Node_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.Node",
	HandlerType: (*NodeServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Init",
			Handler:    _Node_Init_Handler,
		},
		{
			MethodName: "OnCreate",
			Handler:    _Node_OnCreate_Handler,
		},
		{
			MethodName: "OnMessage",
			Handler:    _Node_OnMessage_Handler,
		},
		{
			MethodName: "OnClose",
			Handler:    _Node_OnClose_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "plugin.proto",
}

// RuntimeHelperClient is the client API for RuntimeHelper service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RuntimeHelperClient interface {
	Close(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error)
	Debug(ctx context.Context, in *DebugRequest, opts ...grpc.CallOption) (*Empty, error)
	EmitFlowEvent(ctx context.Context, in *EmitFlowEventRequest, opts ...grpc.CallOption) (*Empty, error)
	EmitInput(ctx context.Context, in *EmitInputRequest, opts ...grpc.CallOption) (*Empty, error)
	EmitOutput(ctx context.Context, in *EmitOutputRequest, opts ...grpc.CallOption) (*Empty, error)
	EmitError(ctx context.Context, in *EmitErrorRequest, opts ...grpc.CallOption) (*Empty, error)
	GetVaultItem(ctx context.Context, in *GetVaultItemRequest, opts ...grpc.CallOption) (*GetVaultItemResponse, error)
	SetVaultItem(ctx context.Context, in *SetVaultItemRequest, opts ...grpc.CallOption) (*SetVaultItemResponse, error)
	GetVariable(ctx context.Context, in *GetVariableRequest, opts ...grpc.CallOption) (*GetVariableResponse, error)
	SetVariable(ctx context.Context, in *SetVariableRequest, opts ...grpc.CallOption) (*Empty, error)
	GetRobotInfo(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetRobotInfoResponse, error)
	AppRequest(ctx context.Context, in *AppRequestRequest, opts ...grpc.CallOption) (*AppRequestResponse, error)
}

type runtimeHelperClient struct {
	cc grpc.ClientConnInterface
}

func NewRuntimeHelperClient(cc grpc.ClientConnInterface) RuntimeHelperClient {
	return &runtimeHelperClient{cc}
}

func (c *runtimeHelperClient) Close(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/Close", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) Debug(ctx context.Context, in *DebugRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/Debug", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) EmitFlowEvent(ctx context.Context, in *EmitFlowEventRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/EmitFlowEvent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) EmitInput(ctx context.Context, in *EmitInputRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/EmitInput", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) EmitOutput(ctx context.Context, in *EmitOutputRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/EmitOutput", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) EmitError(ctx context.Context, in *EmitErrorRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/EmitError", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) GetVaultItem(ctx context.Context, in *GetVaultItemRequest, opts ...grpc.CallOption) (*GetVaultItemResponse, error) {
	out := new(GetVaultItemResponse)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/GetVaultItem", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) SetVaultItem(ctx context.Context, in *SetVaultItemRequest, opts ...grpc.CallOption) (*SetVaultItemResponse, error) {
	out := new(SetVaultItemResponse)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/SetVaultItem", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) GetVariable(ctx context.Context, in *GetVariableRequest, opts ...grpc.CallOption) (*GetVariableResponse, error) {
	out := new(GetVariableResponse)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/GetVariable", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) SetVariable(ctx context.Context, in *SetVariableRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/SetVariable", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) GetRobotInfo(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetRobotInfoResponse, error) {
	out := new(GetRobotInfoResponse)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/GetRobotInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHelperClient) AppRequest(ctx context.Context, in *AppRequestRequest, opts ...grpc.CallOption) (*AppRequestResponse, error) {
	out := new(AppRequestResponse)
	err := c.cc.Invoke(ctx, "/proto.RuntimeHelper/AppRequest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RuntimeHelperServer is the server API for RuntimeHelper service.
// All implementations must embed UnimplementedRuntimeHelperServer
// for forward compatibility
type RuntimeHelperServer interface {
	Close(context.Context, *Empty) (*Empty, error)
	Debug(context.Context, *DebugRequest) (*Empty, error)
	EmitFlowEvent(context.Context, *EmitFlowEventRequest) (*Empty, error)
	EmitInput(context.Context, *EmitInputRequest) (*Empty, error)
	EmitOutput(context.Context, *EmitOutputRequest) (*Empty, error)
	EmitError(context.Context, *EmitErrorRequest) (*Empty, error)
	GetVaultItem(context.Context, *GetVaultItemRequest) (*GetVaultItemResponse, error)
	SetVaultItem(context.Context, *SetVaultItemRequest) (*SetVaultItemResponse, error)
	GetVariable(context.Context, *GetVariableRequest) (*GetVariableResponse, error)
	SetVariable(context.Context, *SetVariableRequest) (*Empty, error)
	GetRobotInfo(context.Context, *Empty) (*GetRobotInfoResponse, error)
	AppRequest(context.Context, *AppRequestRequest) (*AppRequestResponse, error)
	mustEmbedUnimplementedRuntimeHelperServer()
}

// UnimplementedRuntimeHelperServer must be embedded to have forward compatible implementations.
type UnimplementedRuntimeHelperServer struct {
}

func (UnimplementedRuntimeHelperServer) Close(context.Context, *Empty) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Close not implemented")
}
func (UnimplementedRuntimeHelperServer) Debug(context.Context, *DebugRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Debug not implemented")
}
func (UnimplementedRuntimeHelperServer) EmitFlowEvent(context.Context, *EmitFlowEventRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EmitFlowEvent not implemented")
}
func (UnimplementedRuntimeHelperServer) EmitInput(context.Context, *EmitInputRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EmitInput not implemented")
}
func (UnimplementedRuntimeHelperServer) EmitOutput(context.Context, *EmitOutputRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EmitOutput not implemented")
}
func (UnimplementedRuntimeHelperServer) EmitError(context.Context, *EmitErrorRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EmitError not implemented")
}
func (UnimplementedRuntimeHelperServer) GetVaultItem(context.Context, *GetVaultItemRequest) (*GetVaultItemResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVaultItem not implemented")
}
func (UnimplementedRuntimeHelperServer) SetVaultItem(context.Context, *SetVaultItemRequest) (*SetVaultItemResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetVaultItem not implemented")
}
func (UnimplementedRuntimeHelperServer) GetVariable(context.Context, *GetVariableRequest) (*GetVariableResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVariable not implemented")
}
func (UnimplementedRuntimeHelperServer) SetVariable(context.Context, *SetVariableRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetVariable not implemented")
}
func (UnimplementedRuntimeHelperServer) GetRobotInfo(context.Context, *Empty) (*GetRobotInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRobotInfo not implemented")
}
func (UnimplementedRuntimeHelperServer) AppRequest(context.Context, *AppRequestRequest) (*AppRequestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AppRequest not implemented")
}
func (UnimplementedRuntimeHelperServer) mustEmbedUnimplementedRuntimeHelperServer() {}

// UnsafeRuntimeHelperServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RuntimeHelperServer will
// result in compilation errors.
type UnsafeRuntimeHelperServer interface {
	mustEmbedUnimplementedRuntimeHelperServer()
}

func RegisterRuntimeHelperServer(s grpc.ServiceRegistrar, srv RuntimeHelperServer) {
	s.RegisterService(&RuntimeHelper_ServiceDesc, srv)
}

func _RuntimeHelper_Close_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).Close(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/Close",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).Close(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_Debug_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DebugRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).Debug(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/Debug",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).Debug(ctx, req.(*DebugRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_EmitFlowEvent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmitFlowEventRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).EmitFlowEvent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/EmitFlowEvent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).EmitFlowEvent(ctx, req.(*EmitFlowEventRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_EmitInput_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmitInputRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).EmitInput(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/EmitInput",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).EmitInput(ctx, req.(*EmitInputRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_EmitOutput_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmitOutputRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).EmitOutput(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/EmitOutput",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).EmitOutput(ctx, req.(*EmitOutputRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_EmitError_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmitErrorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).EmitError(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/EmitError",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).EmitError(ctx, req.(*EmitErrorRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_GetVaultItem_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetVaultItemRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).GetVaultItem(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/GetVaultItem",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).GetVaultItem(ctx, req.(*GetVaultItemRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_SetVaultItem_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetVaultItemRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).SetVaultItem(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/SetVaultItem",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).SetVaultItem(ctx, req.(*SetVaultItemRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_GetVariable_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetVariableRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).GetVariable(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/GetVariable",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).GetVariable(ctx, req.(*GetVariableRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_SetVariable_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetVariableRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).SetVariable(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/SetVariable",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).SetVariable(ctx, req.(*SetVariableRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_GetRobotInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).GetRobotInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/GetRobotInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).GetRobotInfo(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHelper_AppRequest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AppRequestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHelperServer).AppRequest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.RuntimeHelper/AppRequest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHelperServer).AppRequest(ctx, req.(*AppRequestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RuntimeHelper_ServiceDesc is the grpc.ServiceDesc for RuntimeHelper service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RuntimeHelper_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.RuntimeHelper",
	HandlerType: (*RuntimeHelperServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Close",
			Handler:    _RuntimeHelper_Close_Handler,
		},
		{
			MethodName: "Debug",
			Handler:    _RuntimeHelper_Debug_Handler,
		},
		{
			MethodName: "EmitFlowEvent",
			Handler:    _RuntimeHelper_EmitFlowEvent_Handler,
		},
		{
			MethodName: "EmitInput",
			Handler:    _RuntimeHelper_EmitInput_Handler,
		},
		{
			MethodName: "EmitOutput",
			Handler:    _RuntimeHelper_EmitOutput_Handler,
		},
		{
			MethodName: "EmitError",
			Handler:    _RuntimeHelper_EmitError_Handler,
		},
		{
			MethodName: "GetVaultItem",
			Handler:    _RuntimeHelper_GetVaultItem_Handler,
		},
		{
			MethodName: "SetVaultItem",
			Handler:    _RuntimeHelper_SetVaultItem_Handler,
		},
		{
			MethodName: "GetVariable",
			Handler:    _RuntimeHelper_GetVariable_Handler,
		},
		{
			MethodName: "SetVariable",
			Handler:    _RuntimeHelper_SetVariable_Handler,
		},
		{
			MethodName: "GetRobotInfo",
			Handler:    _RuntimeHelper_GetRobotInfo_Handler,
		},
		{
			MethodName: "AppRequest",
			Handler:    _RuntimeHelper_AppRequest_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "plugin.proto",
}
