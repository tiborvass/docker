// Code generated by protoc-gen-go. DO NOT EDIT.
// source: multilog.proto

/*
Package configpb is a generated protocol buffer package.

It is generated from these files:
	multilog.proto

It has these top-level messages:
	TemporalLogConfig
	LogShardConfig
*/
package configpb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// TemporalLogConfig is a set of LogShardConfig messages, whose
// time limits should be contiguous.
type TemporalLogConfig struct {
	Shard []*LogShardConfig `protobuf:"bytes,1,rep,name=shard" json:"shard,omitempty"`
}

func (m *TemporalLogConfig) Reset()                    { *m = TemporalLogConfig{} }
func (m *TemporalLogConfig) String() string            { return proto.CompactTextString(m) }
func (*TemporalLogConfig) ProtoMessage()               {}
func (*TemporalLogConfig) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *TemporalLogConfig) GetShard() []*LogShardConfig {
	if m != nil {
		return m.Shard
	}
	return nil
}

// LogShardConfig describes the acceptable date range for a single shard of a temporal
// log.
type LogShardConfig struct {
	Uri string `protobuf:"bytes,1,opt,name=uri" json:"uri,omitempty"`
	// The log's public key in DER-encoded PKIX form.
	PublicKeyDer []byte `protobuf:"bytes,2,opt,name=public_key_der,json=publicKeyDer,proto3" json:"public_key_der,omitempty"`
	// not_after_start defines the start of the range of acceptable NotAfter
	// values, inclusive.
	// Leaving this unset implies no lower bound to the range.
	NotAfterStart *google_protobuf.Timestamp `protobuf:"bytes,3,opt,name=not_after_start,json=notAfterStart" json:"not_after_start,omitempty"`
	// not_after_limit defines the end of the range of acceptable NotAfter values,
	// exclusive.
	// Leaving this unset implies no upper bound to the range.
	NotAfterLimit *google_protobuf.Timestamp `protobuf:"bytes,4,opt,name=not_after_limit,json=notAfterLimit" json:"not_after_limit,omitempty"`
}

func (m *LogShardConfig) Reset()                    { *m = LogShardConfig{} }
func (m *LogShardConfig) String() string            { return proto.CompactTextString(m) }
func (*LogShardConfig) ProtoMessage()               {}
func (*LogShardConfig) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *LogShardConfig) GetUri() string {
	if m != nil {
		return m.Uri
	}
	return ""
}

func (m *LogShardConfig) GetPublicKeyDer() []byte {
	if m != nil {
		return m.PublicKeyDer
	}
	return nil
}

func (m *LogShardConfig) GetNotAfterStart() *google_protobuf.Timestamp {
	if m != nil {
		return m.NotAfterStart
	}
	return nil
}

func (m *LogShardConfig) GetNotAfterLimit() *google_protobuf.Timestamp {
	if m != nil {
		return m.NotAfterLimit
	}
	return nil
}

func init() {
	proto.RegisterType((*TemporalLogConfig)(nil), "configpb.TemporalLogConfig")
	proto.RegisterType((*LogShardConfig)(nil), "configpb.LogShardConfig")
}

func init() { proto.RegisterFile("multilog.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 241 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x8f, 0xb1, 0x4e, 0xc3, 0x30,
	0x14, 0x45, 0x65, 0x02, 0x08, 0xdc, 0x12, 0xc0, 0x93, 0xd5, 0x85, 0xa8, 0x62, 0xc8, 0xe4, 0x4a,
	0xe5, 0x0b, 0xa0, 0x6c, 0x64, 0x4a, 0xbb, 0x47, 0x4e, 0xeb, 0x18, 0x0b, 0x3b, 0xcf, 0x72, 0x5e,
	0x86, 0xfe, 0x25, 0x9f, 0x84, 0x1c, 0x2b, 0x43, 0x37, 0xb6, 0xa7, 0x77, 0xcf, 0xb9, 0xd2, 0xa5,
	0xb9, 0x1b, 0x2d, 0x1a, 0x0b, 0x5a, 0xf8, 0x00, 0x08, 0xec, 0xee, 0x08, 0x7d, 0x67, 0xb4, 0x6f,
	0x57, 0x2f, 0x1a, 0x40, 0x5b, 0xb5, 0x99, 0xfe, 0xed, 0xd8, 0x6d, 0xd0, 0x38, 0x35, 0xa0, 0x74,
	0x3e, 0xa1, 0xeb, 0x1d, 0x7d, 0x3e, 0x28, 0xe7, 0x21, 0x48, 0x5b, 0x81, 0xde, 0x4d, 0x1e, 0x13,
	0xf4, 0x66, 0xf8, 0x96, 0xe1, 0xc4, 0x49, 0x91, 0x95, 0x8b, 0x2d, 0x17, 0x73, 0x9f, 0xa8, 0x40,
	0xef, 0x63, 0x92, 0xc0, 0x3a, 0x61, 0xeb, 0x5f, 0x42, 0xf3, 0xcb, 0x84, 0x3d, 0xd1, 0x6c, 0x0c,
	0x86, 0x93, 0x82, 0x94, 0xf7, 0x75, 0x3c, 0xd9, 0x2b, 0xcd, 0xfd, 0xd8, 0x5a, 0x73, 0x6c, 0x7e,
	0xd4, 0xb9, 0x39, 0xa9, 0xc0, 0xaf, 0x0a, 0x52, 0x2e, 0xeb, 0x65, 0xfa, 0x7e, 0xa9, 0xf3, 0xa7,
	0x0a, 0xec, 0x83, 0x3e, 0xf6, 0x80, 0x8d, 0xec, 0x50, 0x85, 0x66, 0x40, 0x19, 0x90, 0x67, 0x05,
	0x29, 0x17, 0xdb, 0x95, 0x48, 0x53, 0xc4, 0x3c, 0x45, 0x1c, 0xe6, 0x29, 0xf5, 0x43, 0x0f, 0xf8,
	0x1e, 0x8d, 0x7d, 0x14, 0x2e, 0x3b, 0xac, 0x71, 0x06, 0xf9, 0xf5, 0xff, 0x3b, 0xaa, 0x28, 0xb4,
	0xb7, 0x13, 0xf2, 0xf6, 0x17, 0x00, 0x00, 0xff, 0xff, 0xf8, 0xd9, 0x50, 0x5b, 0x5b, 0x01, 0x00,
	0x00,
}
