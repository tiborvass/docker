// cgo -godefs types_aix.go | go run mkpost.go
// Code generated by the command above; see README.md. DO NOT EDIT.

// +build ppc64,aix

package unix

const (
	sizeofPtr      = 0x8
	sizeofShort    = 0x2
	sizeofInt      = 0x4
	sizeofLong     = 0x8
	sizeofLongLong = 0x8
	PathMax        = 0x3ff
)

type (
	_C_short     int16
	_C_int       int32
	_C_long      int64
	_C_long_long int64
)

type off64 int64
type off int64
type Mode_t uint32

type Timespec struct {
	Sec  int64
	Nsec int64
}

type StTimespec struct {
	Sec  int64
	Nsec int32
	_    [4]byte
}

type Timeval struct {
	Sec  int64
	Usec int32
	_    [4]byte
}

type Timeval32 struct {
	Sec  int32
	Usec int32
}

type Timex struct{}

type Time_t int64

type Tms struct{}

type Utimbuf struct {
	Actime  int64
	Modtime int64
}

type Timezone struct {
	Minuteswest int32
	Dsttime     int32
}

type Rusage struct {
	Utime    Timeval
	Stime    Timeval
	Maxrss   int64
	Ixrss    int64
	Idrss    int64
	Isrss    int64
	Minflt   int64
	Majflt   int64
	Nswap    int64
	Inblock  int64
	Oublock  int64
	Msgsnd   int64
	Msgrcv   int64
	Nsignals int64
	Nvcsw    int64
	Nivcsw   int64
}

type Rlimit struct {
	Cur uint64
	Max uint64
}

type Pid_t int32

type _Gid_t uint32

type dev_t uint64

type Stat_t struct {
	Dev      uint64
	Ino      uint64
	Mode     uint32
	Nlink    int16
	Flag     uint16
	Uid      uint32
	Gid      uint32
	Rdev     uint64
	Ssize    int32
	_        [4]byte
	Atim     StTimespec
	Mtim     StTimespec
	Ctim     StTimespec
	Blksize  int64
	Blocks   int64
	Vfstype  int32
	Vfs      uint32
	Type     uint32
	Gen      uint32
	Reserved [9]uint32
	Padto_ll uint32
	Size     int64
}

type StatxTimestamp struct{}

type Statx_t struct{}

type Dirent struct {
	Offset uint64
	Ino    uint64
	Reclen uint16
	Namlen uint16
	Name   [256]uint8
	_      [4]byte
}

type RawSockaddrInet4 struct {
	Len    uint8
	Family uint8
	Port   uint16
	Addr   [4]byte /* in_addr */
	Zero   [8]uint8
}

type RawSockaddrInet6 struct {
	Len      uint8
	Family   uint8
	Port     uint16
	Flowinfo uint32
	Addr     [16]byte /* in6_addr */
	Scope_id uint32
}

type RawSockaddrUnix struct {
	Len    uint8
	Family uint8
	Path   [1023]uint8
}

type RawSockaddr struct {
	Len    uint8
	Family uint8
	Data   [14]uint8
}

type RawSockaddrAny struct {
	Addr RawSockaddr
	Pad  [1012]uint8
}

type _Socklen uint32

type Cmsghdr struct {
	Len   uint32
	Level int32
	Type  int32
}

type ICMPv6Filter struct {
	Filt [8]uint32
}

type Iovec struct {
	Base *byte
	Len  uint64
}

type IPMreq struct {
	Multiaddr [4]byte /* in_addr */
	Interface [4]byte /* in_addr */
}

type IPv6Mreq struct {
	Multiaddr [16]byte /* in6_addr */
	Interface uint32
}

type IPv6MTUInfo struct {
	Addr RawSockaddrInet6
	Mtu  uint32
}

type Linger struct {
	Onoff  int32
	Linger int32
}

type Msghdr struct {
	Name       *byte
	Namelen    uint32
	_          [4]byte
	Iov        *Iovec
	Iovlen     int32
	_          [4]byte
	Control    *byte
	Controllen uint32
	Flags      int32
}

const (
	SizeofSockaddrInet4 = 0x10
	SizeofSockaddrInet6 = 0x1c
	SizeofSockaddrAny   = 0x404
	SizeofSockaddrUnix  = 0x401
	SizeofLinger        = 0x8
	SizeofIPMreq        = 0x8
	SizeofIPv6Mreq      = 0x14
	SizeofIPv6MTUInfo   = 0x20
	SizeofMsghdr        = 0x30
	SizeofCmsghdr       = 0xc
	SizeofICMPv6Filter  = 0x20
)

const (
	SizeofIfMsghdr = 0x10
)

type IfMsgHdr struct {
	Msglen  uint16
	Version uint8
	Type    uint8
	Addrs   int32
	Flags   int32
	Index   uint16
	Addrlen uint8
	_       [1]byte
}

type FdSet struct {
	Bits [1024]int64
}

type Utsname struct {
	Sysname  [32]byte
	Nodename [32]byte
	Release  [32]byte
	Version  [32]byte
	Machine  [32]byte
}

type Ustat_t struct{}

type Sigset_t struct {
	Set [4]uint64
}

const (
	AT_FDCWD            = -0x2
	AT_REMOVEDIR        = 0x1
	AT_SYMLINK_NOFOLLOW = 0x1
)

type Termios struct {
	Iflag uint32
	Oflag uint32
	Cflag uint32
	Lflag uint32
	Cc    [16]uint8
}

type Termio struct {
	Iflag uint16
	Oflag uint16
	Cflag uint16
	Lflag uint16
	Line  uint8
	Cc    [8]uint8
	_     [1]byte
}

type Winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

type PollFd struct {
	Fd      int32
	Events  uint16
	Revents uint16
}

const (
	POLLERR    = 0x4000
	POLLHUP    = 0x2000
	POLLIN     = 0x1
	POLLNVAL   = 0x8000
	POLLOUT    = 0x2
	POLLPRI    = 0x4
	POLLRDBAND = 0x20
	POLLRDNORM = 0x10
	POLLWRBAND = 0x40
	POLLWRNORM = 0x2
)

type Flock_t struct {
	Type   int16
	Whence int16
	Sysid  uint32
	Pid    int32
	Vfs    int32
	Start  int64
	Len    int64
}

type Fsid_t struct {
	Val [2]uint32
}
type Fsid64_t struct {
	Val [2]uint64
}

type Statfs_t struct {
	Version   int32
	Type      int32
	Bsize     uint64
	Blocks    uint64
	Bfree     uint64
	Bavail    uint64
	Files     uint64
	Ffree     uint64
	Fsid      Fsid64_t
	Vfstype   int32
	_         [4]byte
	Fsize     uint64
	Vfsnumber int32
	Vfsoff    int32
	Vfslen    int32
	Vfsvers   int32
	Fname     [32]uint8
	Fpack     [32]uint8
	Name_max  int32
	_         [4]byte
}

const RNDGETENTCNT = 0x80045200
