// +build !cgo

package copy

import "unsafe"

var _FICLONE uintptr = _IOW(0x94, 9)

func _IOW(typ, nr uintptr) uintptr {
	return _IOC(_IOC_WRITE, typ, nr, unsafe.Sizeof(typ))
}

const (
	_IOC_NRSHIFT = iota * 8
	_IOC_TYPESHIFT
	_IOC_SIZESHIFT
	_IOC_DIRSHIFT
)

const _IOC_WRITE = 1

func _IOC(dir, typ, nr, sz uintptr) uintptr {
	return	(dir << _IOC_DIRSHIFT) |
		(typ << _IOC_TYPESHIFT) |
		(nr << _IOC_NRSHIFT) |
		(sz << _IOC_SIZESHIFT)
}
