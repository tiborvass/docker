// Code generated by protoc-gen-go.
// source: github.com/golang/protobuf/ptypes/timestamp/timestamp.proto
// DO NOT EDIT!

/*
Package timestamp is a generated protocol buffer package.

It is generated from these files:
	github.com/golang/protobuf/ptypes/timestamp/timestamp.proto

It has these top-level messages:
	Timestamp
*/
package timestamp

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// A Timestamp represents a point in time independent of any time zone
// or calendar, represented as seconds and fractions of seconds at
// nanosecond resolution in UTC Epoch time. It is encoded using the
// Proleptic Gregorian Calendar which extends the Gregorian calendar
// backwards to year one. It is encoded assuming all minutes are 60
// seconds long, i.e. leap seconds are "smeared" so that no leap second
// table is needed for interpretation. Range is from
// 0001-01-01T00:00:00Z to 9999-12-31T23:59:59.999999999Z.
// By restricting to that range, we ensure that we can convert to
// and from  RFC 3339 date strings.
// See [https://www.ietf.org/rfc/rfc3339.txt](https://www.ietf.org/rfc/rfc3339.txt).
//
// Example 1: Compute Timestamp from POSIX `time()`.
//
//     Timestamp timestamp;
//     timestamp.set_seconds(time(NULL));
//     timestamp.set_nanos(0);
//
// Example 2: Compute Timestamp from POSIX `gettimeofday()`.
//
//     struct timeval tv;
//     gettimeofday(&tv, NULL);
//
//     Timestamp timestamp;
//     timestamp.set_seconds(tv.tv_sec);
//     timestamp.set_nanos(tv.tv_usec * 1000);
//
// Example 3: Compute Timestamp from Win32 `GetSystemTimeAsFileTime()`.
//
//     FILETIME ft;
//     GetSystemTimeAsFileTime(&ft);
//     UINT64 ticks = (((UINT64)ft.dwHighDateTime) << 32) | ft.dwLowDateTime;
//
//     // A Windows tick is 100 nanoseconds. Windows epoch 1601-01-01T00:00:00Z
//     // is 11644473600 seconds before Unix epoch 1970-01-01T00:00:00Z.
//     Timestamp timestamp;
//     timestamp.set_seconds((INT64) ((ticks / 10000000) - 11644473600LL));
//     timestamp.set_nanos((INT32) ((ticks % 10000000) * 100));
//
// Example 4: Compute Timestamp from Java `System.currentTimeMillis()`.
//
//     long millis = System.currentTimeMillis();
//
//     Timestamp timestamp = Timestamp.newBuilder().setSeconds(millis / 1000)
//         .setNanos((int) ((millis % 1000) * 1000000)).build();
//
//
// Example 5: Compute Timestamp from current time in Python.
//
//     now = time.time()
//     seconds = int(now)
//     nanos = int((now - seconds) * 10**9)
//     timestamp = Timestamp(seconds=seconds, nanos=nanos)
//
//
type Timestamp struct {
	// Represents seconds of UTC time since Unix epoch
	// 1970-01-01T00:00:00Z. Must be from from 0001-01-01T00:00:00Z to
	// 9999-12-31T23:59:59Z inclusive.
	Seconds int64 `protobuf:"varint,1,opt,name=seconds" json:"seconds,omitempty"`
	// Non-negative fractions of a second at nanosecond resolution. Negative
	// second values with fractions must still have non-negative nanos values
	// that count forward in time. Must be from 0 to 999,999,999
	// inclusive.
	Nanos int32 `protobuf:"varint,2,opt,name=nanos" json:"nanos,omitempty"`
}

func (m *Timestamp) Reset()                    { *m = Timestamp{} }
func (m *Timestamp) String() string            { return proto.CompactTextString(m) }
func (*Timestamp) ProtoMessage()               {}
func (*Timestamp) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }
func (*Timestamp) XXX_WellKnownType() string   { return "Timestamp" }

func init() {
	proto.RegisterType((*Timestamp)(nil), "google.protobuf.Timestamp")
}

func init() {
	proto.RegisterFile("github.com/golang/protobuf/ptypes/timestamp/timestamp.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 194 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xb2, 0x4e, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0x4f, 0xcf, 0xcf, 0x49, 0xcc, 0x4b, 0xd7, 0x2f, 0x28,
	0xca, 0x2f, 0xc9, 0x4f, 0x2a, 0x4d, 0xd3, 0x2f, 0x28, 0xa9, 0x2c, 0x48, 0x2d, 0xd6, 0x2f, 0xc9,
	0xcc, 0x4d, 0x2d, 0x2e, 0x49, 0xcc, 0x2d, 0x40, 0xb0, 0xf4, 0xc0, 0x6a, 0x84, 0xf8, 0xd3, 0xf3,
	0xf3, 0xd3, 0x73, 0x52, 0xf5, 0x60, 0x3a, 0x94, 0xac, 0xb9, 0x38, 0x43, 0x60, 0x6a, 0x84, 0x24,
	0xb8, 0xd8, 0x8b, 0x53, 0x93, 0xf3, 0xf3, 0x52, 0x8a, 0x25, 0x18, 0x15, 0x18, 0x35, 0x98, 0x83,
	0x60, 0x5c, 0x21, 0x11, 0x2e, 0xd6, 0xbc, 0xc4, 0xbc, 0xfc, 0x62, 0x09, 0x26, 0x05, 0x46, 0x0d,
	0xd6, 0x20, 0x08, 0xc7, 0xa9, 0x91, 0x91, 0x4b, 0x38, 0x39, 0x3f, 0x57, 0x0f, 0xcd, 0x50, 0x27,
	0x3e, 0xb8, 0x91, 0x01, 0x20, 0xa1, 0x00, 0xc6, 0x28, 0x6d, 0x12, 0x1c, 0xbd, 0x80, 0x91, 0xf1,
	0x07, 0x23, 0xe3, 0x22, 0x26, 0x66, 0xf7, 0x00, 0xa7, 0x55, 0x4c, 0x72, 0xee, 0x10, 0xc3, 0x03,
	0xa0, 0xca, 0xf5, 0xc2, 0x53, 0x73, 0x72, 0xbc, 0xf3, 0xf2, 0xcb, 0xf3, 0x42, 0x40, 0xda, 0x92,
	0xd8, 0xc0, 0xe6, 0x18, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x17, 0x5f, 0xb7, 0xdc, 0x17, 0x01,
	0x00, 0x00,
}
