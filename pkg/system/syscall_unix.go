// +build linux freebsd

package system // import "github.com/tiborvass/docker/pkg/system"

import "golang.org/x/sys/unix"

// Unmount is a platform-specific helper function to call
// the unmount syscall.
func Unmount(dest string) error {
	return unix.Unmount(dest, 0)
}
