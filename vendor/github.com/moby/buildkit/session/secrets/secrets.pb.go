// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: secrets.proto

/*
	Package secrets is a generated protocol buffer package.

	It is generated from these files:
		secrets.proto

	It has these top-level messages:
		GetSecretRequest
		GetSecretResponse
*/
package secrets

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

import bytes "bytes"

import strings "strings"
import reflect "reflect"
import sortkeys "github.com/gogo/protobuf/sortkeys"

import context "golang.org/x/net/context"
import grpc "google.golang.org/grpc"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type GetSecretRequest struct {
	ID          string            `protobuf:"bytes,1,opt,name=ID,proto3" json:"ID,omitempty"`
	Annotations map[string]string `protobuf:"bytes,2,rep,name=annotations" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (m *GetSecretRequest) Reset()                    { *m = GetSecretRequest{} }
func (*GetSecretRequest) ProtoMessage()               {}
func (*GetSecretRequest) Descriptor() ([]byte, []int) { return fileDescriptorSecrets, []int{0} }

func (m *GetSecretRequest) GetID() string {
	if m != nil {
		return m.ID
	}
	return ""
}

func (m *GetSecretRequest) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

type GetSecretResponse struct {
	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (m *GetSecretResponse) Reset()                    { *m = GetSecretResponse{} }
func (*GetSecretResponse) ProtoMessage()               {}
func (*GetSecretResponse) Descriptor() ([]byte, []int) { return fileDescriptorSecrets, []int{1} }

func (m *GetSecretResponse) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
	proto.RegisterType((*GetSecretRequest)(nil), "moby.buildkit.secrets.v1.GetSecretRequest")
	proto.RegisterType((*GetSecretResponse)(nil), "moby.buildkit.secrets.v1.GetSecretResponse")
}
func (this *GetSecretRequest) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*GetSecretRequest)
	if !ok {
		that2, ok := that.(GetSecretRequest)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.ID != that1.ID {
		return false
	}
	if len(this.Annotations) != len(that1.Annotations) {
		return false
	}
	for i := range this.Annotations {
		if this.Annotations[i] != that1.Annotations[i] {
			return false
		}
	}
	return true
}
func (this *GetSecretResponse) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*GetSecretResponse)
	if !ok {
		that2, ok := that.(GetSecretResponse)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if !bytes.Equal(this.Data, that1.Data) {
		return false
	}
	return true
}
func (this *GetSecretRequest) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&secrets.GetSecretRequest{")
	s = append(s, "ID: "+fmt.Sprintf("%#v", this.ID)+",\n")
	keysForAnnotations := make([]string, 0, len(this.Annotations))
	for k, _ := range this.Annotations {
		keysForAnnotations = append(keysForAnnotations, k)
	}
	sortkeys.Strings(keysForAnnotations)
	mapStringForAnnotations := "map[string]string{"
	for _, k := range keysForAnnotations {
		mapStringForAnnotations += fmt.Sprintf("%#v: %#v,", k, this.Annotations[k])
	}
	mapStringForAnnotations += "}"
	if this.Annotations != nil {
		s = append(s, "Annotations: "+mapStringForAnnotations+",\n")
	}
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *GetSecretResponse) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&secrets.GetSecretResponse{")
	s = append(s, "Data: "+fmt.Sprintf("%#v", this.Data)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringSecrets(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Secrets service

type SecretsClient interface {
	GetSecret(ctx context.Context, in *GetSecretRequest, opts ...grpc.CallOption) (*GetSecretResponse, error)
}

type secretsClient struct {
	cc *grpc.ClientConn
}

func NewSecretsClient(cc *grpc.ClientConn) SecretsClient {
	return &secretsClient{cc}
}

func (c *secretsClient) GetSecret(ctx context.Context, in *GetSecretRequest, opts ...grpc.CallOption) (*GetSecretResponse, error) {
	out := new(GetSecretResponse)
	err := grpc.Invoke(ctx, "/moby.buildkit.secrets.v1.Secrets/GetSecret", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Secrets service

type SecretsServer interface {
	GetSecret(context.Context, *GetSecretRequest) (*GetSecretResponse, error)
}

func RegisterSecretsServer(s *grpc.Server, srv SecretsServer) {
	s.RegisterService(&_Secrets_serviceDesc, srv)
}

func _Secrets_GetSecret_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSecretRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SecretsServer).GetSecret(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/moby.buildkit.secrets.v1.Secrets/GetSecret",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SecretsServer).GetSecret(ctx, req.(*GetSecretRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Secrets_serviceDesc = grpc.ServiceDesc{
	ServiceName: "moby.buildkit.secrets.v1.Secrets",
	HandlerType: (*SecretsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetSecret",
			Handler:    _Secrets_GetSecret_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "secrets.proto",
}

func (m *GetSecretRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GetSecretRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.ID) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintSecrets(dAtA, i, uint64(len(m.ID)))
		i += copy(dAtA[i:], m.ID)
	}
	if len(m.Annotations) > 0 {
		for k, _ := range m.Annotations {
			dAtA[i] = 0x12
			i++
			v := m.Annotations[k]
			mapSize := 1 + len(k) + sovSecrets(uint64(len(k))) + 1 + len(v) + sovSecrets(uint64(len(v)))
			i = encodeVarintSecrets(dAtA, i, uint64(mapSize))
			dAtA[i] = 0xa
			i++
			i = encodeVarintSecrets(dAtA, i, uint64(len(k)))
			i += copy(dAtA[i:], k)
			dAtA[i] = 0x12
			i++
			i = encodeVarintSecrets(dAtA, i, uint64(len(v)))
			i += copy(dAtA[i:], v)
		}
	}
	return i, nil
}

func (m *GetSecretResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GetSecretResponse) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Data) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintSecrets(dAtA, i, uint64(len(m.Data)))
		i += copy(dAtA[i:], m.Data)
	}
	return i, nil
}

func encodeVarintSecrets(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *GetSecretRequest) Size() (n int) {
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovSecrets(uint64(l))
	}
	if len(m.Annotations) > 0 {
		for k, v := range m.Annotations {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovSecrets(uint64(len(k))) + 1 + len(v) + sovSecrets(uint64(len(v)))
			n += mapEntrySize + 1 + sovSecrets(uint64(mapEntrySize))
		}
	}
	return n
}

func (m *GetSecretResponse) Size() (n int) {
	var l int
	_ = l
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovSecrets(uint64(l))
	}
	return n
}

func sovSecrets(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozSecrets(x uint64) (n int) {
	return sovSecrets(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *GetSecretRequest) String() string {
	if this == nil {
		return "nil"
	}
	keysForAnnotations := make([]string, 0, len(this.Annotations))
	for k, _ := range this.Annotations {
		keysForAnnotations = append(keysForAnnotations, k)
	}
	sortkeys.Strings(keysForAnnotations)
	mapStringForAnnotations := "map[string]string{"
	for _, k := range keysForAnnotations {
		mapStringForAnnotations += fmt.Sprintf("%v: %v,", k, this.Annotations[k])
	}
	mapStringForAnnotations += "}"
	s := strings.Join([]string{`&GetSecretRequest{`,
		`ID:` + fmt.Sprintf("%v", this.ID) + `,`,
		`Annotations:` + mapStringForAnnotations + `,`,
		`}`,
	}, "")
	return s
}
func (this *GetSecretResponse) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&GetSecretResponse{`,
		`Data:` + fmt.Sprintf("%v", this.Data) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringSecrets(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *GetSecretRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSecrets
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: GetSecretRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GetSecretRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSecrets
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSecrets
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Annotations", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSecrets
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSecrets
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Annotations == nil {
				m.Annotations = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowSecrets
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowSecrets
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= (uint64(b) & 0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthSecrets
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowSecrets
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= (uint64(b) & 0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthSecrets
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipSecrets(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if skippy < 0 {
						return ErrInvalidLengthSecrets
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Annotations[mapkey] = mapvalue
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSecrets(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSecrets
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
func (m *GetSecretResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSecrets
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: GetSecretResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GetSecretResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSecrets
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSecrets
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSecrets(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSecrets
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
func skipSecrets(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSecrets
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
					return 0, ErrIntOverflowSecrets
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
					return 0, ErrIntOverflowSecrets
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
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthSecrets
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowSecrets
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
				next, err := skipSecrets(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
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
	ErrInvalidLengthSecrets = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSecrets   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("secrets.proto", fileDescriptorSecrets) }

var fileDescriptorSecrets = []byte{
	// 279 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2d, 0x4e, 0x4d, 0x2e,
	0x4a, 0x2d, 0x29, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x92, 0xc8, 0xcd, 0x4f, 0xaa, 0xd4,
	0x4b, 0x2a, 0xcd, 0xcc, 0x49, 0xc9, 0xce, 0x2c, 0xd1, 0x83, 0x49, 0x96, 0x19, 0x2a, 0x1d, 0x64,
	0xe4, 0x12, 0x70, 0x4f, 0x2d, 0x09, 0x06, 0x8b, 0x04, 0xa5, 0x16, 0x96, 0xa6, 0x16, 0x97, 0x08,
	0xf1, 0x71, 0x31, 0x79, 0xba, 0x48, 0x30, 0x2a, 0x30, 0x6a, 0x70, 0x06, 0x31, 0x79, 0xba, 0x08,
	0xc5, 0x72, 0x71, 0x27, 0xe6, 0xe5, 0xe5, 0x97, 0x24, 0x96, 0x64, 0xe6, 0xe7, 0x15, 0x4b, 0x30,
	0x29, 0x30, 0x6b, 0x70, 0x1b, 0x59, 0xeb, 0xe1, 0x32, 0x54, 0x0f, 0xdd, 0x40, 0x3d, 0x47, 0x84,
	0x6e, 0xd7, 0xbc, 0x92, 0xa2, 0xca, 0x20, 0x64, 0xf3, 0xa4, 0xec, 0xb8, 0x04, 0xd0, 0x15, 0x08,
	0x09, 0x70, 0x31, 0x67, 0xa7, 0x56, 0x42, 0xdd, 0x00, 0x62, 0x0a, 0x89, 0x70, 0xb1, 0x96, 0x25,
	0xe6, 0x94, 0xa6, 0x4a, 0x30, 0x81, 0xc5, 0x20, 0x1c, 0x2b, 0x26, 0x0b, 0x46, 0x25, 0x75, 0x2e,
	0x41, 0x24, 0x1b, 0x8b, 0x0b, 0xf2, 0xf3, 0x8a, 0x53, 0x85, 0x84, 0xb8, 0x58, 0x52, 0x12, 0x4b,
	0x12, 0xc1, 0x26, 0xf0, 0x04, 0x81, 0xd9, 0x46, 0xf9, 0x5c, 0xec, 0x10, 0x55, 0xc5, 0x42, 0x29,
	0x5c, 0x9c, 0x70, 0x3d, 0x42, 0x5a, 0xc4, 0x7b, 0x45, 0x4a, 0x9b, 0x28, 0xb5, 0x10, 0x47, 0x38,
	0x99, 0x5e, 0x78, 0x28, 0xc7, 0x70, 0xe3, 0xa1, 0x1c, 0xc3, 0x87, 0x87, 0x72, 0x8c, 0x0d, 0x8f,
	0xe4, 0x18, 0x57, 0x3c, 0x92, 0x63, 0x3c, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07,
	0x8f, 0xe4, 0x18, 0x5f, 0x3c, 0x92, 0x63, 0xf8, 0xf0, 0x48, 0x8e, 0x71, 0xc2, 0x63, 0x39, 0x86,
	0x28, 0x76, 0xa8, 0x59, 0x49, 0x6c, 0xe0, 0x58, 0x33, 0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x05,
	0x4e, 0x56, 0xde, 0xc6, 0x01, 0x00, 0x00,
}
