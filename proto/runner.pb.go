// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.12.4
// source: runner.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Null struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Null) Reset() {
	*x = Null{}
	if protoimpl.UnsafeEnabled {
		mi := &file_runner_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Null) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Null) ProtoMessage() {}

func (x *Null) ProtoReflect() protoreflect.Message {
	mi := &file_runner_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Null.ProtoReflect.Descriptor instead.
func (*Null) Descriptor() ([]byte, []int) {
	return file_runner_proto_rawDescGZIP(), []int{0}
}

type InitRunnerRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ApiUrl      string `protobuf:"bytes,1,opt,name=apiUrl,proto3" json:"apiUrl,omitempty"`
	Token       string `protobuf:"bytes,2,opt,name=token,proto3" json:"token,omitempty"`
	RobotServer uint32 `protobuf:"varint,3,opt,name=robot_server,json=robotServer,proto3" json:"robot_server,omitempty"`
	NatsPort    uint32 `protobuf:"varint,4,opt,name=natsPort,proto3" json:"natsPort,omitempty"`
	RobotName   string `protobuf:"bytes,5,opt,name=robot_name,json=robotName,proto3" json:"robot_name,omitempty"`
	SendCrash   bool   `protobuf:"varint,6,opt,name=send_crash,json=sendCrash,proto3" json:"send_crash,omitempty"`
}

func (x *InitRunnerRequest) Reset() {
	*x = InitRunnerRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_runner_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InitRunnerRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InitRunnerRequest) ProtoMessage() {}

func (x *InitRunnerRequest) ProtoReflect() protoreflect.Message {
	mi := &file_runner_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InitRunnerRequest.ProtoReflect.Descriptor instead.
func (*InitRunnerRequest) Descriptor() ([]byte, []int) {
	return file_runner_proto_rawDescGZIP(), []int{1}
}

func (x *InitRunnerRequest) GetApiUrl() string {
	if x != nil {
		return x.ApiUrl
	}
	return ""
}

func (x *InitRunnerRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *InitRunnerRequest) GetRobotServer() uint32 {
	if x != nil {
		return x.RobotServer
	}
	return 0
}

func (x *InitRunnerRequest) GetNatsPort() uint32 {
	if x != nil {
		return x.NatsPort
	}
	return 0
}

func (x *InitRunnerRequest) GetRobotName() string {
	if x != nil {
		return x.RobotName
	}
	return ""
}

func (x *InitRunnerRequest) GetSendCrash() bool {
	if x != nil {
		return x.SendCrash
	}
	return false
}

type RunRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Repoid     string `protobuf:"bytes,1,opt,name=repoid,proto3" json:"repoid,omitempty"`
	Repository string `protobuf:"bytes,2,opt,name=repository,proto3" json:"repository,omitempty"`
	Namespace  string `protobuf:"bytes,3,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Version    string `protobuf:"bytes,4,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *RunRequest) Reset() {
	*x = RunRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_runner_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunRequest) ProtoMessage() {}

func (x *RunRequest) ProtoReflect() protoreflect.Message {
	mi := &file_runner_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RunRequest.ProtoReflect.Descriptor instead.
func (*RunRequest) Descriptor() ([]byte, []int) {
	return file_runner_proto_rawDescGZIP(), []int{2}
}

func (x *RunRequest) GetRepoid() string {
	if x != nil {
		return x.Repoid
	}
	return ""
}

func (x *RunRequest) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *RunRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *RunRequest) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

type RobotNameResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RobotName string `protobuf:"bytes,1,opt,name=robot_name,json=robotName,proto3" json:"robot_name,omitempty"`
}

func (x *RobotNameResponse) Reset() {
	*x = RobotNameResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_runner_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RobotNameResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RobotNameResponse) ProtoMessage() {}

func (x *RobotNameResponse) ProtoReflect() protoreflect.Message {
	mi := &file_runner_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RobotNameResponse.ProtoReflect.Descriptor instead.
func (*RobotNameResponse) Descriptor() ([]byte, []int) {
	return file_runner_proto_rawDescGZIP(), []int{3}
}

func (x *RobotNameResponse) GetRobotName() string {
	if x != nil {
		return x.RobotName
	}
	return ""
}

type GetCapabilitiesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Capabilities uint64 `protobuf:"varint,1,opt,name=capabilities,proto3" json:"capabilities,omitempty"`
}

func (x *GetCapabilitiesResponse) Reset() {
	*x = GetCapabilitiesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_runner_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCapabilitiesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCapabilitiesResponse) ProtoMessage() {}

func (x *GetCapabilitiesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_runner_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCapabilitiesResponse.ProtoReflect.Descriptor instead.
func (*GetCapabilitiesResponse) Descriptor() ([]byte, []int) {
	return file_runner_proto_rawDescGZIP(), []int{4}
}

func (x *GetCapabilitiesResponse) GetCapabilities() uint64 {
	if x != nil {
		return x.Capabilities
	}
	return 0
}

type Nil struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Nil) Reset() {
	*x = Nil{}
	if protoimpl.UnsafeEnabled {
		mi := &file_runner_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Nil) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Nil) ProtoMessage() {}

func (x *Nil) ProtoReflect() protoreflect.Message {
	mi := &file_runner_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Nil.ProtoReflect.Descriptor instead.
func (*Nil) Descriptor() ([]byte, []int) {
	return file_runner_proto_rawDescGZIP(), []int{5}
}

type AttachRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config []byte `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *AttachRequest) Reset() {
	*x = AttachRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_runner_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttachRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttachRequest) ProtoMessage() {}

func (x *AttachRequest) ProtoReflect() protoreflect.Message {
	mi := &file_runner_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttachRequest.ProtoReflect.Descriptor instead.
func (*AttachRequest) Descriptor() ([]byte, []int) {
	return file_runner_proto_rawDescGZIP(), []int{6}
}

func (x *AttachRequest) GetConfig() []byte {
	if x != nil {
		return x.Config
	}
	return nil
}

type DetachRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
}

func (x *DetachRequest) Reset() {
	*x = DetachRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_runner_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DetachRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DetachRequest) ProtoMessage() {}

func (x *DetachRequest) ProtoReflect() protoreflect.Message {
	mi := &file_runner_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DetachRequest.ProtoReflect.Descriptor instead.
func (*DetachRequest) Descriptor() ([]byte, []int) {
	return file_runner_proto_rawDescGZIP(), []int{7}
}

func (x *DetachRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

var File_runner_proto protoreflect.FileDescriptor

var file_runner_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x72, 0x75, 0x6e, 0x6e, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x06, 0x0a, 0x04, 0x4e, 0x75, 0x6c, 0x6c, 0x22, 0xbe, 0x01,
	0x0a, 0x11, 0x49, 0x6e, 0x69, 0x74, 0x52, 0x75, 0x6e, 0x6e, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x70, 0x69, 0x55, 0x72, 0x6c, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x61, 0x70, 0x69, 0x55, 0x72, 0x6c, 0x12, 0x14, 0x0a, 0x05, 0x74,
	0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65,
	0x6e, 0x12, 0x21, 0x0a, 0x0c, 0x72, 0x6f, 0x62, 0x6f, 0x74, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0b, 0x72, 0x6f, 0x62, 0x6f, 0x74, 0x53, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x12, 0x1a, 0x0a, 0x08, 0x6e, 0x61, 0x74, 0x73, 0x50, 0x6f, 0x72, 0x74,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x6e, 0x61, 0x74, 0x73, 0x50, 0x6f, 0x72, 0x74,
	0x12, 0x1d, 0x0a, 0x0a, 0x72, 0x6f, 0x62, 0x6f, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x72, 0x6f, 0x62, 0x6f, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12,
	0x1d, 0x0a, 0x0a, 0x73, 0x65, 0x6e, 0x64, 0x5f, 0x63, 0x72, 0x61, 0x73, 0x68, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x09, 0x73, 0x65, 0x6e, 0x64, 0x43, 0x72, 0x61, 0x73, 0x68, 0x22, 0x7c,
	0x0a, 0x0a, 0x52, 0x75, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x0a, 0x06,
	0x72, 0x65, 0x70, 0x6f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65,
	0x70, 0x6f, 0x69, 0x64, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0x32, 0x0a, 0x11,
	0x52, 0x6f, 0x62, 0x6f, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x72, 0x6f, 0x62, 0x6f, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x72, 0x6f, 0x62, 0x6f, 0x74, 0x4e, 0x61, 0x6d, 0x65,
	0x22, 0x3d, 0x0a, 0x17, 0x47, 0x65, 0x74, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74,
	0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x22, 0x0a, 0x0c, 0x63,
	0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x0c, 0x63, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65, 0x73, 0x22,
	0x05, 0x0a, 0x03, 0x4e, 0x69, 0x6c, 0x22, 0x27, 0x0a, 0x0d, 0x41, 0x74, 0x74, 0x61, 0x63, 0x68,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22,
	0x2d, 0x0a, 0x0d, 0x44, 0x65, 0x74, 0x61, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x32, 0xf5,
	0x01, 0x0a, 0x06, 0x52, 0x75, 0x6e, 0x6e, 0x65, 0x72, 0x12, 0x2d, 0x0a, 0x04, 0x49, 0x6e, 0x69,
	0x74, 0x12, 0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x49, 0x6e, 0x69, 0x74, 0x52, 0x75,
	0x6e, 0x6e, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0b, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x4e, 0x75, 0x6c, 0x6c, 0x12, 0x25, 0x0a, 0x03, 0x52, 0x75, 0x6e, 0x12,
	0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x75, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x0b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4e, 0x75, 0x6c, 0x6c, 0x12,
	0x21, 0x0a, 0x05, 0x43, 0x6c, 0x65, 0x61, 0x72, 0x12, 0x0b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x4e, 0x75, 0x6c, 0x6c, 0x1a, 0x0b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4e, 0x75,
	0x6c, 0x6c, 0x12, 0x32, 0x0a, 0x09, 0x52, 0x6f, 0x62, 0x6f, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12,
	0x0b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4e, 0x75, 0x6c, 0x6c, 0x1a, 0x18, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x6f, 0x62, 0x6f, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3e, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x43, 0x61, 0x70,
	0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65, 0x73, 0x12, 0x0b, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x4e, 0x75, 0x6c, 0x6c, 0x1a, 0x1e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47,
	0x65, 0x74, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x0d, 0x0a, 0x0b, 0x52, 0x6f, 0x62, 0x6f, 0x74, 0x48,
	0x65, 0x6c, 0x70, 0x65, 0x72, 0x32, 0x5f, 0x0a, 0x05, 0x44, 0x65, 0x62, 0x75, 0x67, 0x12, 0x2a,
	0x0a, 0x06, 0x41, 0x74, 0x74, 0x61, 0x63, 0x68, 0x12, 0x14, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x41, 0x74, 0x74, 0x61, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0a,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4e, 0x69, 0x6c, 0x12, 0x2a, 0x0a, 0x06, 0x44, 0x65,
	0x74, 0x61, 0x63, 0x68, 0x12, 0x14, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x44, 0x65, 0x74,
	0x61, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0a, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x4e, 0x69, 0x6c, 0x42, 0x2a, 0x5a, 0x28, 0x72, 0x6f, 0x62, 0x6f, 0x6d, 0x6f,
	0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x72, 0x6f, 0x62, 0x6f, 0x6d, 0x6f, 0x74, 0x69, 0x6f, 0x6e, 0x2d,
	0x72, 0x75, 0x6e, 0x6e, 0x65, 0x72, 0x2f, 0x72, 0x6f, 0x62, 0x6f, 0x74, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_runner_proto_rawDescOnce sync.Once
	file_runner_proto_rawDescData = file_runner_proto_rawDesc
)

func file_runner_proto_rawDescGZIP() []byte {
	file_runner_proto_rawDescOnce.Do(func() {
		file_runner_proto_rawDescData = protoimpl.X.CompressGZIP(file_runner_proto_rawDescData)
	})
	return file_runner_proto_rawDescData
}

var file_runner_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_runner_proto_goTypes = []interface{}{
	(*Null)(nil),                    // 0: proto.Null
	(*InitRunnerRequest)(nil),       // 1: proto.InitRunnerRequest
	(*RunRequest)(nil),              // 2: proto.RunRequest
	(*RobotNameResponse)(nil),       // 3: proto.RobotNameResponse
	(*GetCapabilitiesResponse)(nil), // 4: proto.GetCapabilitiesResponse
	(*Nil)(nil),                     // 5: proto.Nil
	(*AttachRequest)(nil),           // 6: proto.AttachRequest
	(*DetachRequest)(nil),           // 7: proto.DetachRequest
}
var file_runner_proto_depIdxs = []int32{
	1, // 0: proto.Runner.Init:input_type -> proto.InitRunnerRequest
	2, // 1: proto.Runner.Run:input_type -> proto.RunRequest
	0, // 2: proto.Runner.Clear:input_type -> proto.Null
	0, // 3: proto.Runner.RobotName:input_type -> proto.Null
	0, // 4: proto.Runner.GetCapabilities:input_type -> proto.Null
	6, // 5: proto.Debug.Attach:input_type -> proto.AttachRequest
	7, // 6: proto.Debug.Detach:input_type -> proto.DetachRequest
	0, // 7: proto.Runner.Init:output_type -> proto.Null
	0, // 8: proto.Runner.Run:output_type -> proto.Null
	0, // 9: proto.Runner.Clear:output_type -> proto.Null
	3, // 10: proto.Runner.RobotName:output_type -> proto.RobotNameResponse
	4, // 11: proto.Runner.GetCapabilities:output_type -> proto.GetCapabilitiesResponse
	5, // 12: proto.Debug.Attach:output_type -> proto.Nil
	5, // 13: proto.Debug.Detach:output_type -> proto.Nil
	7, // [7:14] is the sub-list for method output_type
	0, // [0:7] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_runner_proto_init() }
func file_runner_proto_init() {
	if File_runner_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_runner_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Null); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_runner_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InitRunnerRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_runner_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RunRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_runner_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RobotNameResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_runner_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCapabilitiesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_runner_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Nil); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_runner_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttachRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_runner_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DetachRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_runner_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   3,
		},
		GoTypes:           file_runner_proto_goTypes,
		DependencyIndexes: file_runner_proto_depIdxs,
		MessageInfos:      file_runner_proto_msgTypes,
	}.Build()
	File_runner_proto = out.File
	file_runner_proto_rawDesc = nil
	file_runner_proto_goTypes = nil
	file_runner_proto_depIdxs = nil
}
