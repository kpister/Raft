// Code generated by protoc-gen-go. DO NOT EDIT.
// source: kvstore.proto

package kvstore

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// You'll likely need to define more specific return codes than these!
type ReturnCode int32

const (
	ReturnCode_SUCCESS ReturnCode = 0
	ReturnCode_FAILURE ReturnCode = 1
)

var ReturnCode_name = map[int32]string{
	0: "SUCCESS",
	1: "FAILURE",
}

var ReturnCode_value = map[string]int32{
	"SUCCESS": 0,
	"FAILURE": 1,
}

func (x ReturnCode) String() string {
	return proto.EnumName(ReturnCode_name, int32(x))
}

func (ReturnCode) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_088d7f6aff848d9e, []int{0}
}

type GetRequest struct {
	Key                  string   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetRequest) Reset()         { *m = GetRequest{} }
func (m *GetRequest) String() string { return proto.CompactTextString(m) }
func (*GetRequest) ProtoMessage()    {}
func (*GetRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_088d7f6aff848d9e, []int{0}
}

func (m *GetRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetRequest.Unmarshal(m, b)
}
func (m *GetRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetRequest.Marshal(b, m, deterministic)
}
func (m *GetRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetRequest.Merge(m, src)
}
func (m *GetRequest) XXX_Size() int {
	return xxx_messageInfo_GetRequest.Size(m)
}
func (m *GetRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetRequest proto.InternalMessageInfo

func (m *GetRequest) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

type GetResponse struct {
	Value                string     `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	Ret                  ReturnCode `protobuf:"varint,2,opt,name=ret,proto3,enum=kvstore.ReturnCode" json:"ret,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *GetResponse) Reset()         { *m = GetResponse{} }
func (m *GetResponse) String() string { return proto.CompactTextString(m) }
func (*GetResponse) ProtoMessage()    {}
func (*GetResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_088d7f6aff848d9e, []int{1}
}

func (m *GetResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetResponse.Unmarshal(m, b)
}
func (m *GetResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetResponse.Marshal(b, m, deterministic)
}
func (m *GetResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetResponse.Merge(m, src)
}
func (m *GetResponse) XXX_Size() int {
	return xxx_messageInfo_GetResponse.Size(m)
}
func (m *GetResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetResponse proto.InternalMessageInfo

func (m *GetResponse) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

func (m *GetResponse) GetRet() ReturnCode {
	if m != nil {
		return m.Ret
	}
	return ReturnCode_SUCCESS
}

type PutRequest struct {
	Key                  string   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value                string   `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	From                 int32    `protobuf:"varint,3,opt,name=from,proto3" json:"from,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PutRequest) Reset()         { *m = PutRequest{} }
func (m *PutRequest) String() string { return proto.CompactTextString(m) }
func (*PutRequest) ProtoMessage()    {}
func (*PutRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_088d7f6aff848d9e, []int{2}
}

func (m *PutRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PutRequest.Unmarshal(m, b)
}
func (m *PutRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PutRequest.Marshal(b, m, deterministic)
}
func (m *PutRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PutRequest.Merge(m, src)
}
func (m *PutRequest) XXX_Size() int {
	return xxx_messageInfo_PutRequest.Size(m)
}
func (m *PutRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_PutRequest.DiscardUnknown(m)
}

var xxx_messageInfo_PutRequest proto.InternalMessageInfo

func (m *PutRequest) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *PutRequest) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

func (m *PutRequest) GetFrom() int32 {
	if m != nil {
		return m.From
	}
	return 0
}

type PutResponse struct {
	Ret                  ReturnCode `protobuf:"varint,1,opt,name=ret,proto3,enum=kvstore.ReturnCode" json:"ret,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *PutResponse) Reset()         { *m = PutResponse{} }
func (m *PutResponse) String() string { return proto.CompactTextString(m) }
func (*PutResponse) ProtoMessage()    {}
func (*PutResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_088d7f6aff848d9e, []int{3}
}

func (m *PutResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PutResponse.Unmarshal(m, b)
}
func (m *PutResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PutResponse.Marshal(b, m, deterministic)
}
func (m *PutResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PutResponse.Merge(m, src)
}
func (m *PutResponse) XXX_Size() int {
	return xxx_messageInfo_PutResponse.Size(m)
}
func (m *PutResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_PutResponse.DiscardUnknown(m)
}

var xxx_messageInfo_PutResponse proto.InternalMessageInfo

func (m *PutResponse) GetRet() ReturnCode {
	if m != nil {
		return m.Ret
	}
	return ReturnCode_SUCCESS
}

func init() {
	proto.RegisterEnum("kvstore.ReturnCode", ReturnCode_name, ReturnCode_value)
	proto.RegisterType((*GetRequest)(nil), "kvstore.GetRequest")
	proto.RegisterType((*GetResponse)(nil), "kvstore.GetResponse")
	proto.RegisterType((*PutRequest)(nil), "kvstore.PutRequest")
	proto.RegisterType((*PutResponse)(nil), "kvstore.PutResponse")
}

func init() { proto.RegisterFile("kvstore.proto", fileDescriptor_088d7f6aff848d9e) }

var fileDescriptor_088d7f6aff848d9e = []byte{
	// 254 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0x3f, 0x4f, 0xc3, 0x30,
	0x10, 0xc5, 0xe3, 0x86, 0x52, 0xf1, 0xa2, 0xa2, 0xc8, 0x74, 0x88, 0x18, 0x50, 0x64, 0x09, 0x14,
	0x31, 0x74, 0x08, 0x7c, 0x01, 0x14, 0x95, 0xf2, 0x6f, 0xa8, 0x1c, 0x95, 0x1d, 0xc4, 0xb1, 0x14,
	0xea, 0xe2, 0xd8, 0x45, 0xfd, 0xf6, 0xc8, 0x49, 0x90, 0xcd, 0x40, 0xb7, 0x7b, 0x77, 0x79, 0xef,
	0x77, 0x17, 0x63, 0xbc, 0xda, 0x36, 0x46, 0x69, 0x9a, 0x6e, 0xb4, 0x32, 0x8a, 0x8f, 0x7a, 0x29,
	0xce, 0x80, 0x39, 0x19, 0x49, 0x5f, 0x96, 0x1a, 0xc3, 0x53, 0xc4, 0x2b, 0xda, 0x65, 0x2c, 0x67,
	0xc5, 0x91, 0x74, 0xa5, 0x78, 0x40, 0xd2, 0xce, 0x9b, 0x8d, 0x5a, 0x37, 0xc4, 0x27, 0x18, 0x6e,
	0x5f, 0x3e, 0x2c, 0xf5, 0x9f, 0x74, 0x82, 0x9f, 0x23, 0xd6, 0x64, 0xb2, 0x41, 0xce, 0x8a, 0xe3,
	0xf2, 0x64, 0xfa, 0x8b, 0x92, 0x64, 0xac, 0x5e, 0x57, 0xea, 0x8d, 0xa4, 0x9b, 0x8b, 0x3b, 0x60,
	0x61, 0xff, 0x67, 0xf9, 0xf0, 0x41, 0x18, 0xce, 0x71, 0xf0, 0xae, 0xd5, 0x67, 0x16, 0xe7, 0xac,
	0x18, 0xca, 0xb6, 0x16, 0xd7, 0x48, 0xda, 0xa4, 0x7e, 0xab, 0x9e, 0xcf, 0xf6, 0xf3, 0x2f, 0x2f,
	0x00, 0xdf, 0xe2, 0x09, 0x46, 0xf5, 0xb2, 0xaa, 0x66, 0x75, 0x9d, 0x46, 0x4e, 0xdc, 0xde, 0xdc,
	0x3f, 0x2d, 0xe5, 0x2c, 0x65, 0xe5, 0x37, 0xc6, 0x8f, 0xb4, 0x7b, 0x76, 0xf4, 0xda, 0x05, 0xf1,
	0x12, 0xf1, 0x9c, 0x0c, 0xf7, 0xc9, 0xfe, 0x97, 0x9d, 0x4e, 0xfe, 0x36, 0xbb, 0x8d, 0x44, 0xe4,
	0x3c, 0x0b, 0x1b, 0x7a, 0xfc, 0xe9, 0x81, 0x27, 0xb8, 0x42, 0x44, 0xaf, 0x87, 0xed, 0xe3, 0x5c,
	0xfd, 0x04, 0x00, 0x00, 0xff, 0xff, 0x88, 0xb8, 0x7f, 0x7d, 0xad, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// KeyValueStoreClient is the client API for KeyValueStore service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type KeyValueStoreClient interface {
	Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error)
	Put(ctx context.Context, in *PutRequest, opts ...grpc.CallOption) (*PutResponse, error)
}

type keyValueStoreClient struct {
	cc *grpc.ClientConn
}

func NewKeyValueStoreClient(cc *grpc.ClientConn) KeyValueStoreClient {
	return &keyValueStoreClient{cc}
}

func (c *keyValueStoreClient) Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error) {
	out := new(GetResponse)
	err := c.cc.Invoke(ctx, "/kvstore.KeyValueStore/Get", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *keyValueStoreClient) Put(ctx context.Context, in *PutRequest, opts ...grpc.CallOption) (*PutResponse, error) {
	out := new(PutResponse)
	err := c.cc.Invoke(ctx, "/kvstore.KeyValueStore/Put", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// KeyValueStoreServer is the server API for KeyValueStore service.
type KeyValueStoreServer interface {
	Get(context.Context, *GetRequest) (*GetResponse, error)
	Put(context.Context, *PutRequest) (*PutResponse, error)
}

func RegisterKeyValueStoreServer(s *grpc.Server, srv KeyValueStoreServer) {
	s.RegisterService(&_KeyValueStore_serviceDesc, srv)
}

func _KeyValueStore_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KeyValueStoreServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kvstore.KeyValueStore/Get",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KeyValueStoreServer).Get(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KeyValueStore_Put_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PutRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KeyValueStoreServer).Put(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kvstore.KeyValueStore/Put",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KeyValueStoreServer).Put(ctx, req.(*PutRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _KeyValueStore_serviceDesc = grpc.ServiceDesc{
	ServiceName: "kvstore.KeyValueStore",
	HandlerType: (*KeyValueStoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Get",
			Handler:    _KeyValueStore_Get_Handler,
		},
		{
			MethodName: "Put",
			Handler:    _KeyValueStore_Put_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "kvstore.proto",
}
