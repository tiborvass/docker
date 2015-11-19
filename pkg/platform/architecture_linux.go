// Package platform provides helper function to get the runtime architecture
// for different platforms.
package platform

import (
	"syscall"
)

// GetRuntimeArchitecture get the name of the current architecture (x86, x86_64, …)
func GetRuntimeArchitecture() (string, error) {
	utsname := &syscall.Utsname{}
	if err := syscall.Uname(utsname); err != nil {
		return "", err
	}
	return charsToString(utsname.Machine), nil
}
