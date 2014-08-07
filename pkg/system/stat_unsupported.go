// +build !linux,!windows

package system

import (
	"syscall"
)

type unixTime syscall.Timespec

func (ut unixTime) Unix() int64 {
	return int64(syscall.Timespec(ut).Sec)
}

func (ut unixTime) Nanosecond() int {
	return int(syscall.Timespec(ut).Nsec)
}

func GetLastAccess(stat *syscall.Stat_t) Time {
	return unixTime(stat.Atimespec)
}

func GetLastModification(stat *syscall.Stat_t) Time {
	return unixTime(stat.Mtimespec)
}
