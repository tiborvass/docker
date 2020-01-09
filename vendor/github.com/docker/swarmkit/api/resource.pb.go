// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: github.com/docker/swarmkit/api/resource.proto

package api

import (
	context "context"
	fmt "fmt"
	github_com_docker_swarmkit_api_deepcopy "github.com/docker/swarmkit/api/deepcopy"
	raftselector "github.com/docker/swarmkit/manager/raftselector"
	_ "github.com/docker/swarmkit/protobuf/plugin"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
	peer "google.golang.org/grpc/peer"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	reflect "reflect"
	strings "strings"
	rafttime "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type AttachNetworkRequest struct {
	Config      *NetworkAttachmentConfig `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
	ContainerID string                   `protobuf:"bytes,2,opt,name=container_id,json=containerId,proto3" json:"container_id,omitempty"`
}

func (m *AttachNetworkRequest) Reset()      { *m = AttachNetworkRequest{} }
func (*AttachNetworkRequest) ProtoMessage() {}
func (*AttachNetworkRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_909455b1b868ddb9, []int{0}
}
func (m *AttachNetworkRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AttachNetworkRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AttachNetworkRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AttachNetworkRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AttachNetworkRequest.Merge(m, src)
}
func (m *AttachNetworkRequest) XXX_Size() int {
	return m.Size()
}
func (m *AttachNetworkRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AttachNetworkRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AttachNetworkRequest proto.InternalMessageInfo

type AttachNetworkResponse struct {
	AttachmentID string `protobuf:"bytes,1,opt,name=attachment_id,json=attachmentId,proto3" json:"attachment_id,omitempty"`
}

func (m *AttachNetworkResponse) Reset()      { *m = AttachNetworkResponse{} }
func (*AttachNetworkResponse) ProtoMessage() {}
func (*AttachNetworkResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_909455b1b868ddb9, []int{1}
}
func (m *AttachNetworkResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AttachNetworkResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AttachNetworkResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AttachNetworkResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AttachNetworkResponse.Merge(m, src)
}
func (m *AttachNetworkResponse) XXX_Size() int {
	return m.Size()
}
func (m *AttachNetworkResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_AttachNetworkResponse.DiscardUnknown(m)
}

var xxx_messageInfo_AttachNetworkResponse proto.InternalMessageInfo

type DetachNetworkRequest struct {
	AttachmentID string `protobuf:"bytes,1,opt,name=attachment_id,json=attachmentId,proto3" json:"attachment_id,omitempty"`
}

func (m *DetachNetworkRequest) Reset()      { *m = DetachNetworkRequest{} }
func (*DetachNetworkRequest) ProtoMessage() {}
func (*DetachNetworkRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_909455b1b868ddb9, []int{2}
}
func (m *DetachNetworkRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DetachNetworkRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DetachNetworkRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DetachNetworkRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DetachNetworkRequest.Merge(m, src)
}
func (m *DetachNetworkRequest) XXX_Size() int {
	return m.Size()
}
func (m *DetachNetworkRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DetachNetworkRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DetachNetworkRequest proto.InternalMessageInfo

type DetachNetworkResponse struct {
}

func (m *DetachNetworkResponse) Reset()      { *m = DetachNetworkResponse{} }
func (*DetachNetworkResponse) ProtoMessage() {}
func (*DetachNetworkResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_909455b1b868ddb9, []int{3}
}
func (m *DetachNetworkResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DetachNetworkResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DetachNetworkResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DetachNetworkResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DetachNetworkResponse.Merge(m, src)
}
func (m *DetachNetworkResponse) XXX_Size() int {
	return m.Size()
}
func (m *DetachNetworkResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DetachNetworkResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DetachNetworkResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*AttachNetworkRequest)(nil), "docker.swarmkit.v1.AttachNetworkRequest")
	proto.RegisterType((*AttachNetworkResponse)(nil), "docker.swarmkit.v1.AttachNetworkResponse")
	proto.RegisterType((*DetachNetworkRequest)(nil), "docker.swarmkit.v1.DetachNetworkRequest")
	proto.RegisterType((*DetachNetworkResponse)(nil), "docker.swarmkit.v1.DetachNetworkResponse")
}

func init() {
	proto.RegisterFile("github.com/docker/swarmkit/api/resource.proto", fileDescriptor_909455b1b868ddb9)
}

var fileDescriptor_909455b1b868ddb9 = []byte{
	// 411 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x92, 0x3f, 0x6f, 0xda, 0x40,
	0x18, 0xc6, 0x7d, 0x0c, 0x48, 0x3d, 0x8c, 0xda, 0x5a, 0xa0, 0x22, 0xa4, 0x1e, 0xc8, 0xed, 0x40,
	0x5b, 0x61, 0xab, 0x54, 0x55, 0x67, 0xfe, 0x2c, 0x1e, 0xca, 0xe0, 0x2f, 0x50, 0x1d, 0xf6, 0x61,
	0x2c, 0xb0, 0xcf, 0x3d, 0x9f, 0x83, 0xb2, 0x65, 0xcd, 0x94, 0x8c, 0xf9, 0x0e, 0x91, 0xf2, 0x39,
	0x50, 0x26, 0x46, 0x26, 0x14, 0xcc, 0x9e, 0xcf, 0x10, 0xe1, 0x33, 0x20, 0x88, 0x95, 0xa0, 0x4c,
	0x3e, 0x9f, 0x9f, 0xe7, 0x79, 0x7f, 0xef, 0xfb, 0x1a, 0x36, 0x1d, 0x97, 0x8f, 0xa2, 0x81, 0x66,
	0x51, 0x4f, 0xb7, 0xa9, 0x35, 0x26, 0x4c, 0x0f, 0xa7, 0x98, 0x79, 0x63, 0x97, 0xeb, 0x38, 0x70,
	0x75, 0x46, 0x42, 0x1a, 0x31, 0x8b, 0x68, 0x01, 0xa3, 0x9c, 0x2a, 0x8a, 0xd0, 0x68, 0x5b, 0x8d,
	0x76, 0xf6, 0xb3, 0xfa, 0xfd, 0x95, 0x08, 0x7e, 0x1e, 0x90, 0x50, 0xf8, 0xab, 0x25, 0x87, 0x3a,
	0x34, 0x39, 0xea, 0x9b, 0x53, 0x7a, 0xfb, 0xe7, 0x85, 0x84, 0x44, 0x31, 0x88, 0x86, 0x7a, 0x30,
	0x89, 0x1c, 0xd7, 0x4f, 0x1f, 0xc2, 0xa8, 0x5e, 0x01, 0x58, 0x6a, 0x73, 0x8e, 0xad, 0x51, 0x9f,
	0xf0, 0x29, 0x65, 0x63, 0x93, 0xfc, 0x8f, 0x48, 0xc8, 0x95, 0x2e, 0xcc, 0x5b, 0xd4, 0x1f, 0xba,
	0x4e, 0x05, 0xd4, 0x41, 0xa3, 0xd0, 0xfa, 0xa1, 0x3d, 0x07, 0xd7, 0x52, 0x8f, 0x08, 0xf0, 0x88,
	0xcf, 0xbb, 0x89, 0xc5, 0x4c, 0xad, 0x4a, 0x0b, 0xca, 0x16, 0xf5, 0x39, 0x76, 0x7d, 0xc2, 0xfe,
	0xb9, 0x76, 0x25, 0x57, 0x07, 0x8d, 0x77, 0x9d, 0xf7, 0xf1, 0xb2, 0x56, 0xe8, 0x6e, 0xef, 0x8d,
	0x9e, 0x59, 0xd8, 0x89, 0x0c, 0x5b, 0xed, 0xc3, 0xf2, 0x11, 0x50, 0x18, 0x50, 0x3f, 0x24, 0xca,
	0x6f, 0x58, 0xc4, 0xbb, 0x42, 0x9b, 0x34, 0x90, 0xa4, 0x7d, 0x88, 0x97, 0x35, 0x79, 0x4f, 0x60,
	0xf4, 0x4c, 0x79, 0x2f, 0x33, 0x6c, 0xf5, 0x2f, 0x2c, 0xf5, 0x48, 0x46, 0x83, 0x6f, 0x8c, 0xfb,
	0x04, 0xcb, 0x47, 0x71, 0x02, 0xaf, 0x75, 0x9b, 0x83, 0x1f, 0xcd, 0x74, 0xd7, 0xed, 0xc9, 0x84,
	0x5a, 0x98, 0x53, 0xa6, 0x5c, 0x02, 0x58, 0x3c, 0x68, 0x47, 0x69, 0x64, 0x0d, 0x32, 0x6b, 0x05,
	0xd5, 0x6f, 0x27, 0x28, 0x45, 0x71, 0xf5, 0xcb, 0xfd, 0xdd, 0xe3, 0x4d, 0xee, 0x33, 0x94, 0x13,
	0x69, 0x73, 0xf3, 0x8d, 0x30, 0x58, 0x14, 0x6f, 0x1e, 0xf6, 0xb1, 0x43, 0x04, 0xcb, 0x01, 0x7b,
	0x36, 0x4b, 0xd6, 0xb4, 0xb2, 0x59, 0x32, 0x07, 0x71, 0x12, 0x4b, 0xe7, 0xeb, 0x6c, 0x85, 0xa4,
	0xc5, 0x0a, 0x49, 0x17, 0x31, 0x02, 0xb3, 0x18, 0x81, 0x79, 0x8c, 0xc0, 0x43, 0x8c, 0xc0, 0xf5,
	0x1a, 0x49, 0xf3, 0x35, 0x92, 0x16, 0x6b, 0x24, 0x0d, 0xf2, 0xc9, 0x4f, 0xfa, 0xeb, 0x29, 0x00,
	0x00, 0xff, 0xff, 0x9d, 0x2f, 0x31, 0x83, 0x64, 0x03, 0x00, 0x00,
}

type authenticatedWrapperResourceAllocatorServer struct {
	local     ResourceAllocatorServer
	authorize func(context.Context, []string) error
}

func NewAuthenticatedWrapperResourceAllocatorServer(local ResourceAllocatorServer, authorize func(context.Context, []string) error) ResourceAllocatorServer {
	return &authenticatedWrapperResourceAllocatorServer{
		local:     local,
		authorize: authorize,
	}
}

func (p *authenticatedWrapperResourceAllocatorServer) AttachNetwork(ctx context.Context, r *AttachNetworkRequest) (*AttachNetworkResponse, error) {

	if err := p.authorize(ctx, []string{"swarm-worker", "swarm-manager"}); err != nil {
		return nil, err
	}
	return p.local.AttachNetwork(ctx, r)
}

func (p *authenticatedWrapperResourceAllocatorServer) DetachNetwork(ctx context.Context, r *DetachNetworkRequest) (*DetachNetworkResponse, error) {

	if err := p.authorize(ctx, []string{"swarm-worker", "swarm-manager"}); err != nil {
		return nil, err
	}
	return p.local.DetachNetwork(ctx, r)
}

func (m *AttachNetworkRequest) Copy() *AttachNetworkRequest {
	if m == nil {
		return nil
	}
	o := &AttachNetworkRequest{}
	o.CopyFrom(m)
	return o
}

func (m *AttachNetworkRequest) CopyFrom(src interface{}) {

	o := src.(*AttachNetworkRequest)
	*m = *o
	if o.Config != nil {
		m.Config = &NetworkAttachmentConfig{}
		github_com_docker_swarmkit_api_deepcopy.Copy(m.Config, o.Config)
	}
}

func (m *AttachNetworkResponse) Copy() *AttachNetworkResponse {
	if m == nil {
		return nil
	}
	o := &AttachNetworkResponse{}
	o.CopyFrom(m)
	return o
}

func (m *AttachNetworkResponse) CopyFrom(src interface{}) {

	o := src.(*AttachNetworkResponse)
	*m = *o
}

func (m *DetachNetworkRequest) Copy() *DetachNetworkRequest {
	if m == nil {
		return nil
	}
	o := &DetachNetworkRequest{}
	o.CopyFrom(m)
	return o
}

func (m *DetachNetworkRequest) CopyFrom(src interface{}) {

	o := src.(*DetachNetworkRequest)
	*m = *o
}

func (m *DetachNetworkResponse) Copy() *DetachNetworkResponse {
	if m == nil {
		return nil
	}
	o := &DetachNetworkResponse{}
	o.CopyFrom(m)
	return o
}

func (m *DetachNetworkResponse) CopyFrom(src interface{}) {}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ResourceAllocatorClient is the client API for ResourceAllocator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ResourceAllocatorClient interface {
	AttachNetwork(ctx context.Context, in *AttachNetworkRequest, opts ...grpc.CallOption) (*AttachNetworkResponse, error)
	DetachNetwork(ctx context.Context, in *DetachNetworkRequest, opts ...grpc.CallOption) (*DetachNetworkResponse, error)
}

type resourceAllocatorClient struct {
	cc *grpc.ClientConn
}

func NewResourceAllocatorClient(cc *grpc.ClientConn) ResourceAllocatorClient {
	return &resourceAllocatorClient{cc}
}

func (c *resourceAllocatorClient) AttachNetwork(ctx context.Context, in *AttachNetworkRequest, opts ...grpc.CallOption) (*AttachNetworkResponse, error) {
	out := new(AttachNetworkResponse)
	err := c.cc.Invoke(ctx, "/docker.swarmkit.v1.ResourceAllocator/AttachNetwork", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *resourceAllocatorClient) DetachNetwork(ctx context.Context, in *DetachNetworkRequest, opts ...grpc.CallOption) (*DetachNetworkResponse, error) {
	out := new(DetachNetworkResponse)
	err := c.cc.Invoke(ctx, "/docker.swarmkit.v1.ResourceAllocator/DetachNetwork", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ResourceAllocatorServer is the server API for ResourceAllocator service.
type ResourceAllocatorServer interface {
	AttachNetwork(context.Context, *AttachNetworkRequest) (*AttachNetworkResponse, error)
	DetachNetwork(context.Context, *DetachNetworkRequest) (*DetachNetworkResponse, error)
}

func RegisterResourceAllocatorServer(s *grpc.Server, srv ResourceAllocatorServer) {
	s.RegisterService(&_ResourceAllocator_serviceDesc, srv)
}

func _ResourceAllocator_AttachNetwork_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AttachNetworkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceAllocatorServer).AttachNetwork(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/docker.swarmkit.v1.ResourceAllocator/AttachNetwork",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceAllocatorServer).AttachNetwork(ctx, req.(*AttachNetworkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceAllocator_DetachNetwork_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DetachNetworkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceAllocatorServer).DetachNetwork(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/docker.swarmkit.v1.ResourceAllocator/DetachNetwork",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceAllocatorServer).DetachNetwork(ctx, req.(*DetachNetworkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _ResourceAllocator_serviceDesc = grpc.ServiceDesc{
	ServiceName: "docker.swarmkit.v1.ResourceAllocator",
	HandlerType: (*ResourceAllocatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AttachNetwork",
			Handler:    _ResourceAllocator_AttachNetwork_Handler,
		},
		{
			MethodName: "DetachNetwork",
			Handler:    _ResourceAllocator_DetachNetwork_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "github.com/docker/swarmkit/api/resource.proto",
}

func (m *AttachNetworkRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AttachNetworkRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Config != nil {
		dAtA[i] = 0xa
		i++
		i = encodeVarintResource(dAtA, i, uint64(m.Config.Size()))
		n1, err := m.Config.MarshalTo(dAtA[i:])
		if err != nil {
			return 0, err
		}
		i += n1
	}
	if len(m.ContainerID) > 0 {
		dAtA[i] = 0x12
		i++
		i = encodeVarintResource(dAtA, i, uint64(len(m.ContainerID)))
		i += copy(dAtA[i:], m.ContainerID)
	}
	return i, nil
}

func (m *AttachNetworkResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AttachNetworkResponse) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.AttachmentID) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintResource(dAtA, i, uint64(len(m.AttachmentID)))
		i += copy(dAtA[i:], m.AttachmentID)
	}
	return i, nil
}

func (m *DetachNetworkRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DetachNetworkRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.AttachmentID) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintResource(dAtA, i, uint64(len(m.AttachmentID)))
		i += copy(dAtA[i:], m.AttachmentID)
	}
	return i, nil
}

func (m *DetachNetworkResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DetachNetworkResponse) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	return i, nil
}

func encodeVarintResource(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}

type raftProxyResourceAllocatorServer struct {
	local                       ResourceAllocatorServer
	connSelector                raftselector.ConnProvider
	localCtxMods, remoteCtxMods []func(context.Context) (context.Context, error)
}

func NewRaftProxyResourceAllocatorServer(local ResourceAllocatorServer, connSelector raftselector.ConnProvider, localCtxMod, remoteCtxMod func(context.Context) (context.Context, error)) ResourceAllocatorServer {
	redirectChecker := func(ctx context.Context) (context.Context, error) {
		p, ok := peer.FromContext(ctx)
		if !ok {
			return ctx, status.Errorf(codes.InvalidArgument, "remote addr is not found in context")
		}
		addr := p.Addr.String()
		md, ok := metadata.FromIncomingContext(ctx)
		if ok && len(md["redirect"]) != 0 {
			return ctx, status.Errorf(codes.ResourceExhausted, "more than one redirect to leader from: %s", md["redirect"])
		}
		if !ok {
			md = metadata.New(map[string]string{})
		}
		md["redirect"] = append(md["redirect"], addr)
		return metadata.NewOutgoingContext(ctx, md), nil
	}
	remoteMods := []func(context.Context) (context.Context, error){redirectChecker}
	remoteMods = append(remoteMods, remoteCtxMod)

	var localMods []func(context.Context) (context.Context, error)
	if localCtxMod != nil {
		localMods = []func(context.Context) (context.Context, error){localCtxMod}
	}

	return &raftProxyResourceAllocatorServer{
		local:         local,
		connSelector:  connSelector,
		localCtxMods:  localMods,
		remoteCtxMods: remoteMods,
	}
}
func (p *raftProxyResourceAllocatorServer) runCtxMods(ctx context.Context, ctxMods []func(context.Context) (context.Context, error)) (context.Context, error) {
	var err error
	for _, mod := range ctxMods {
		ctx, err = mod(ctx)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}
func (p *raftProxyResourceAllocatorServer) pollNewLeaderConn(ctx context.Context) (*grpc.ClientConn, error) {
	ticker := rafttime.NewTicker(500 * rafttime.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn, err := p.connSelector.LeaderConn(ctx)
			if err != nil {
				return nil, err
			}

			client := NewHealthClient(conn)

			resp, err := client.Check(ctx, &HealthCheckRequest{Service: "Raft"})
			if err != nil || resp.Status != HealthCheckResponse_SERVING {
				continue
			}
			return conn, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (p *raftProxyResourceAllocatorServer) AttachNetwork(ctx context.Context, r *AttachNetworkRequest) (*AttachNetworkResponse, error) {

	conn, err := p.connSelector.LeaderConn(ctx)
	if err != nil {
		if err == raftselector.ErrIsLeader {
			ctx, err = p.runCtxMods(ctx, p.localCtxMods)
			if err != nil {
				return nil, err
			}
			return p.local.AttachNetwork(ctx, r)
		}
		return nil, err
	}
	modCtx, err := p.runCtxMods(ctx, p.remoteCtxMods)
	if err != nil {
		return nil, err
	}

	resp, err := NewResourceAllocatorClient(conn).AttachNetwork(modCtx, r)
	if err != nil {
		if !strings.Contains(err.Error(), "is closing") && !strings.Contains(err.Error(), "the connection is unavailable") && !strings.Contains(err.Error(), "connection error") {
			return resp, err
		}
		conn, err := p.pollNewLeaderConn(ctx)
		if err != nil {
			if err == raftselector.ErrIsLeader {
				return p.local.AttachNetwork(ctx, r)
			}
			return nil, err
		}
		return NewResourceAllocatorClient(conn).AttachNetwork(modCtx, r)
	}
	return resp, err
}

func (p *raftProxyResourceAllocatorServer) DetachNetwork(ctx context.Context, r *DetachNetworkRequest) (*DetachNetworkResponse, error) {

	conn, err := p.connSelector.LeaderConn(ctx)
	if err != nil {
		if err == raftselector.ErrIsLeader {
			ctx, err = p.runCtxMods(ctx, p.localCtxMods)
			if err != nil {
				return nil, err
			}
			return p.local.DetachNetwork(ctx, r)
		}
		return nil, err
	}
	modCtx, err := p.runCtxMods(ctx, p.remoteCtxMods)
	if err != nil {
		return nil, err
	}

	resp, err := NewResourceAllocatorClient(conn).DetachNetwork(modCtx, r)
	if err != nil {
		if !strings.Contains(err.Error(), "is closing") && !strings.Contains(err.Error(), "the connection is unavailable") && !strings.Contains(err.Error(), "connection error") {
			return resp, err
		}
		conn, err := p.pollNewLeaderConn(ctx)
		if err != nil {
			if err == raftselector.ErrIsLeader {
				return p.local.DetachNetwork(ctx, r)
			}
			return nil, err
		}
		return NewResourceAllocatorClient(conn).DetachNetwork(modCtx, r)
	}
	return resp, err
}

func (m *AttachNetworkRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Config != nil {
		l = m.Config.Size()
		n += 1 + l + sovResource(uint64(l))
	}
	l = len(m.ContainerID)
	if l > 0 {
		n += 1 + l + sovResource(uint64(l))
	}
	return n
}

func (m *AttachNetworkResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.AttachmentID)
	if l > 0 {
		n += 1 + l + sovResource(uint64(l))
	}
	return n
}

func (m *DetachNetworkRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.AttachmentID)
	if l > 0 {
		n += 1 + l + sovResource(uint64(l))
	}
	return n
}

func (m *DetachNetworkResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovResource(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozResource(x uint64) (n int) {
	return sovResource(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *AttachNetworkRequest) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&AttachNetworkRequest{`,
		`Config:` + strings.Replace(fmt.Sprintf("%v", this.Config), "NetworkAttachmentConfig", "NetworkAttachmentConfig", 1) + `,`,
		`ContainerID:` + fmt.Sprintf("%v", this.ContainerID) + `,`,
		`}`,
	}, "")
	return s
}
func (this *AttachNetworkResponse) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&AttachNetworkResponse{`,
		`AttachmentID:` + fmt.Sprintf("%v", this.AttachmentID) + `,`,
		`}`,
	}, "")
	return s
}
func (this *DetachNetworkRequest) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&DetachNetworkRequest{`,
		`AttachmentID:` + fmt.Sprintf("%v", this.AttachmentID) + `,`,
		`}`,
	}, "")
	return s
}
func (this *DetachNetworkResponse) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&DetachNetworkResponse{`,
		`}`,
	}, "")
	return s
}
func valueToStringResource(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *AttachNetworkRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowResource
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: AttachNetworkRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AttachNetworkRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Config", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowResource
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthResource
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthResource
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Config == nil {
				m.Config = &NetworkAttachmentConfig{}
			}
			if err := m.Config.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ContainerID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowResource
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthResource
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthResource
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ContainerID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipResource(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthResource
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthResource
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *AttachNetworkResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowResource
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: AttachNetworkResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AttachNetworkResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AttachmentID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowResource
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthResource
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthResource
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AttachmentID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipResource(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthResource
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthResource
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DetachNetworkRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowResource
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DetachNetworkRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DetachNetworkRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AttachmentID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowResource
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthResource
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthResource
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AttachmentID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipResource(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthResource
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthResource
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DetachNetworkResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowResource
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DetachNetworkResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DetachNetworkResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipResource(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthResource
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthResource
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipResource(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowResource
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowResource
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowResource
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthResource
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthResource
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowResource
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipResource(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthResource
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthResource = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowResource   = fmt.Errorf("proto: integer overflow")
)
