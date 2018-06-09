// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: checksum.proto

/*
	Package contenthash is a generated protocol buffer package.

	It is generated from these files:
		checksum.proto

	It has these top-level messages:
		CacheRecord
		CacheRecordWithPath
		CacheRecords
*/
package contenthash

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gogo/protobuf/gogoproto"

import github_com_opencontainers_go_digest "github.com/opencontainers/go-digest"

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

type CacheRecordType int32

const (
	CacheRecordTypeFile      CacheRecordType = 0
	CacheRecordTypeDir       CacheRecordType = 1
	CacheRecordTypeDirHeader CacheRecordType = 2
	CacheRecordTypeSymlink   CacheRecordType = 3
)

var CacheRecordType_name = map[int32]string{
	0: "FILE",
	1: "DIR",
	2: "DIR_HEADER",
	3: "SYMLINK",
}
var CacheRecordType_value = map[string]int32{
	"FILE":       0,
	"DIR":        1,
	"DIR_HEADER": 2,
	"SYMLINK":    3,
}

func (x CacheRecordType) String() string {
	return proto.EnumName(CacheRecordType_name, int32(x))
}
func (CacheRecordType) EnumDescriptor() ([]byte, []int) { return fileDescriptorChecksum, []int{0} }

type CacheRecord struct {
	Digest   github_com_opencontainers_go_digest.Digest `protobuf:"bytes,1,opt,name=digest,proto3,customtype=github.com/opencontainers/go-digest.Digest" json:"digest"`
	Type     CacheRecordType                            `protobuf:"varint,2,opt,name=type,proto3,enum=contenthash.CacheRecordType" json:"type,omitempty"`
	Linkname string                                     `protobuf:"bytes,3,opt,name=linkname,proto3" json:"linkname,omitempty"`
}

func (m *CacheRecord) Reset()                    { *m = CacheRecord{} }
func (m *CacheRecord) String() string            { return proto.CompactTextString(m) }
func (*CacheRecord) ProtoMessage()               {}
func (*CacheRecord) Descriptor() ([]byte, []int) { return fileDescriptorChecksum, []int{0} }

func (m *CacheRecord) GetType() CacheRecordType {
	if m != nil {
		return m.Type
	}
	return CacheRecordTypeFile
}

func (m *CacheRecord) GetLinkname() string {
	if m != nil {
		return m.Linkname
	}
	return ""
}

type CacheRecordWithPath struct {
	Path   string       `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Record *CacheRecord `protobuf:"bytes,2,opt,name=record" json:"record,omitempty"`
}

func (m *CacheRecordWithPath) Reset()                    { *m = CacheRecordWithPath{} }
func (m *CacheRecordWithPath) String() string            { return proto.CompactTextString(m) }
func (*CacheRecordWithPath) ProtoMessage()               {}
func (*CacheRecordWithPath) Descriptor() ([]byte, []int) { return fileDescriptorChecksum, []int{1} }

func (m *CacheRecordWithPath) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *CacheRecordWithPath) GetRecord() *CacheRecord {
	if m != nil {
		return m.Record
	}
	return nil
}

type CacheRecords struct {
	Paths []*CacheRecordWithPath `protobuf:"bytes,1,rep,name=paths" json:"paths,omitempty"`
}

func (m *CacheRecords) Reset()                    { *m = CacheRecords{} }
func (m *CacheRecords) String() string            { return proto.CompactTextString(m) }
func (*CacheRecords) ProtoMessage()               {}
func (*CacheRecords) Descriptor() ([]byte, []int) { return fileDescriptorChecksum, []int{2} }

func (m *CacheRecords) GetPaths() []*CacheRecordWithPath {
	if m != nil {
		return m.Paths
	}
	return nil
}

func init() {
	proto.RegisterType((*CacheRecord)(nil), "contenthash.CacheRecord")
	proto.RegisterType((*CacheRecordWithPath)(nil), "contenthash.CacheRecordWithPath")
	proto.RegisterType((*CacheRecords)(nil), "contenthash.CacheRecords")
	proto.RegisterEnum("contenthash.CacheRecordType", CacheRecordType_name, CacheRecordType_value)
}
func (m *CacheRecord) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CacheRecord) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Digest) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintChecksum(dAtA, i, uint64(len(m.Digest)))
		i += copy(dAtA[i:], m.Digest)
	}
	if m.Type != 0 {
		dAtA[i] = 0x10
		i++
		i = encodeVarintChecksum(dAtA, i, uint64(m.Type))
	}
	if len(m.Linkname) > 0 {
		dAtA[i] = 0x1a
		i++
		i = encodeVarintChecksum(dAtA, i, uint64(len(m.Linkname)))
		i += copy(dAtA[i:], m.Linkname)
	}
	return i, nil
}

func (m *CacheRecordWithPath) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CacheRecordWithPath) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Path) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintChecksum(dAtA, i, uint64(len(m.Path)))
		i += copy(dAtA[i:], m.Path)
	}
	if m.Record != nil {
		dAtA[i] = 0x12
		i++
		i = encodeVarintChecksum(dAtA, i, uint64(m.Record.Size()))
		n1, err := m.Record.MarshalTo(dAtA[i:])
		if err != nil {
			return 0, err
		}
		i += n1
	}
	return i, nil
}

func (m *CacheRecords) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CacheRecords) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Paths) > 0 {
		for _, msg := range m.Paths {
			dAtA[i] = 0xa
			i++
			i = encodeVarintChecksum(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func encodeVarintChecksum(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *CacheRecord) Size() (n int) {
	var l int
	_ = l
	l = len(m.Digest)
	if l > 0 {
		n += 1 + l + sovChecksum(uint64(l))
	}
	if m.Type != 0 {
		n += 1 + sovChecksum(uint64(m.Type))
	}
	l = len(m.Linkname)
	if l > 0 {
		n += 1 + l + sovChecksum(uint64(l))
	}
	return n
}

func (m *CacheRecordWithPath) Size() (n int) {
	var l int
	_ = l
	l = len(m.Path)
	if l > 0 {
		n += 1 + l + sovChecksum(uint64(l))
	}
	if m.Record != nil {
		l = m.Record.Size()
		n += 1 + l + sovChecksum(uint64(l))
	}
	return n
}

func (m *CacheRecords) Size() (n int) {
	var l int
	_ = l
	if len(m.Paths) > 0 {
		for _, e := range m.Paths {
			l = e.Size()
			n += 1 + l + sovChecksum(uint64(l))
		}
	}
	return n
}

func sovChecksum(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozChecksum(x uint64) (n int) {
	return sovChecksum(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *CacheRecord) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowChecksum
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
			return fmt.Errorf("proto: CacheRecord: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CacheRecord: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Digest", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChecksum
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
				return ErrInvalidLengthChecksum
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Digest = github_com_opencontainers_go_digest.Digest(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChecksum
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= (CacheRecordType(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Linkname", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChecksum
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
				return ErrInvalidLengthChecksum
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Linkname = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipChecksum(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthChecksum
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
func (m *CacheRecordWithPath) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowChecksum
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
			return fmt.Errorf("proto: CacheRecordWithPath: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CacheRecordWithPath: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Path", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChecksum
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
				return ErrInvalidLengthChecksum
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Path = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Record", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChecksum
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
				return ErrInvalidLengthChecksum
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Record == nil {
				m.Record = &CacheRecord{}
			}
			if err := m.Record.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipChecksum(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthChecksum
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
func (m *CacheRecords) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowChecksum
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
			return fmt.Errorf("proto: CacheRecords: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CacheRecords: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Paths", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChecksum
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
				return ErrInvalidLengthChecksum
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Paths = append(m.Paths, &CacheRecordWithPath{})
			if err := m.Paths[len(m.Paths)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipChecksum(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthChecksum
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
func skipChecksum(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowChecksum
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
					return 0, ErrIntOverflowChecksum
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
					return 0, ErrIntOverflowChecksum
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
				return 0, ErrInvalidLengthChecksum
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowChecksum
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
				next, err := skipChecksum(dAtA[start:])
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
	ErrInvalidLengthChecksum = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowChecksum   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("checksum.proto", fileDescriptorChecksum) }

var fileDescriptorChecksum = []byte{
	// 418 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x92, 0xc1, 0x6a, 0xd4, 0x40,
	0x18, 0xc7, 0x77, 0xba, 0xeb, 0xaa, 0xdf, 0x4a, 0x0d, 0x53, 0x68, 0xc3, 0x50, 0xb2, 0xe3, 0x5e,
	0x5c, 0x8a, 0xcd, 0x96, 0x08, 0xde, 0xad, 0xd9, 0xa5, 0xd1, 0x2a, 0x32, 0x15, 0x44, 0x3c, 0x48,
	0x36, 0x3b, 0x66, 0x42, 0x9b, 0x4c, 0x48, 0x66, 0x0f, 0xfb, 0x06, 0x92, 0x93, 0x2f, 0x90, 0x93,
	0x82, 0xef, 0xe0, 0x5d, 0xe8, 0xd1, 0xb3, 0x87, 0x22, 0xeb, 0x8b, 0x48, 0x26, 0x55, 0x42, 0xca,
	0x9e, 0xe6, 0xfb, 0x66, 0x7e, 0xdf, 0xff, 0xff, 0x9f, 0x61, 0x60, 0x3b, 0x10, 0x3c, 0x38, 0xcf,
	0x97, 0xb1, 0x9d, 0x66, 0x52, 0x49, 0x3c, 0x08, 0x64, 0xa2, 0x78, 0xa2, 0x84, 0x9f, 0x0b, 0x72,
	0x18, 0x46, 0x4a, 0x2c, 0xe7, 0x76, 0x20, 0xe3, 0x49, 0x28, 0x43, 0x39, 0xd1, 0xcc, 0x7c, 0xf9,
	0x51, 0x77, 0xba, 0xd1, 0x55, 0x3d, 0x3b, 0xfa, 0x86, 0x60, 0xf0, 0xcc, 0x0f, 0x04, 0x67, 0x3c,
	0x90, 0xd9, 0x02, 0x3f, 0x87, 0xfe, 0x22, 0x0a, 0x79, 0xae, 0x4c, 0x44, 0xd1, 0xf8, 0xee, 0xb1,
	0x73, 0x79, 0x35, 0xec, 0xfc, 0xba, 0x1a, 0x1e, 0x34, 0x64, 0x65, 0xca, 0x93, 0xca, 0xd2, 0x8f,
	0x12, 0x9e, 0xe5, 0x93, 0x50, 0x1e, 0xd6, 0x23, 0xb6, 0xab, 0x17, 0x76, 0xad, 0x80, 0x8f, 0xa0,
	0xa7, 0x56, 0x29, 0x37, 0xb7, 0x28, 0x1a, 0x6f, 0x3b, 0xfb, 0x76, 0x23, 0xa6, 0xdd, 0xf0, 0x7c,
	0xb3, 0x4a, 0x39, 0xd3, 0x24, 0x26, 0x70, 0xe7, 0x22, 0x4a, 0xce, 0x13, 0x3f, 0xe6, 0x66, 0xb7,
	0xf2, 0x67, 0xff, 0xfb, 0xd1, 0x7b, 0xd8, 0x69, 0x0c, 0xbd, 0x8d, 0x94, 0x78, 0xed, 0x2b, 0x81,
	0x31, 0xf4, 0x52, 0x5f, 0x89, 0x3a, 0x2e, 0xd3, 0x35, 0x3e, 0x82, 0x7e, 0xa6, 0x29, 0x6d, 0x3d,
	0x70, 0xcc, 0x4d, 0xd6, 0xec, 0x9a, 0x1b, 0xcd, 0xe0, 0x5e, 0x63, 0x3b, 0xc7, 0x4f, 0xe0, 0x56,
	0xa5, 0x94, 0x9b, 0x88, 0x76, 0xc7, 0x03, 0x87, 0x6e, 0x12, 0xf8, 0x17, 0x83, 0xd5, 0xf8, 0xc1,
	0x0f, 0x04, 0xf7, 0x5b, 0x57, 0xc3, 0x0f, 0xa0, 0x37, 0xf3, 0x4e, 0xa7, 0x46, 0x87, 0xec, 0x15,
	0x25, 0xdd, 0x69, 0x1d, 0xcf, 0xa2, 0x0b, 0x8e, 0x87, 0xd0, 0x75, 0x3d, 0x66, 0x20, 0xb2, 0x5b,
	0x94, 0x14, 0xb7, 0x08, 0x37, 0xca, 0xf0, 0x23, 0x00, 0xd7, 0x63, 0x1f, 0x4e, 0xa6, 0x4f, 0xdd,
	0x29, 0x33, 0xb6, 0xc8, 0x7e, 0x51, 0x52, 0xf3, 0x26, 0x77, 0xc2, 0xfd, 0x05, 0xcf, 0xf0, 0x43,
	0xb8, 0x7d, 0xf6, 0xee, 0xe5, 0xa9, 0xf7, 0xea, 0x85, 0xd1, 0x25, 0xa4, 0x28, 0xe9, 0x6e, 0x0b,
	0x3d, 0x5b, 0xc5, 0xd5, 0xbb, 0x92, 0xbd, 0x4f, 0x5f, 0xac, 0xce, 0xf7, 0xaf, 0x56, 0x3b, 0xf3,
	0xb1, 0x71, 0xb9, 0xb6, 0xd0, 0xcf, 0xb5, 0x85, 0x7e, 0xaf, 0x2d, 0xf4, 0xf9, 0x8f, 0xd5, 0x99,
	0xf7, 0xf5, 0x7f, 0x79, 0xfc, 0x37, 0x00, 0x00, 0xff, 0xff, 0x55, 0xf2, 0x2e, 0x06, 0x7d, 0x02,
	0x00, 0x00,
}
