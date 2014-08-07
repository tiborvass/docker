package system

import "syscall"

type windowsTime struct {
	sec  int64
	nano int
}

func (wt windowsTime) Unix() int64 {
	return wt.sec
}

func (wt windowsTime) Nanosecond() int {
	return wt.nano
}

func newWindowsTime(t syscall.Filetime) windowsTime {
	nano := t.Nanoseconds()
	sec := nano / 1e9
	return windowsTime{sec, int(nano - sec)}
}

func GetLastAccess(stat *syscall.Win32FileAttributeData) Time {
	return newWindowsTime(stat.LastAccessTime)
}

func GetLastModification(stat *syscall.Win32FileAttributeData) Time {
	return newWindowsTime(stat.LastWriteTime)
}
