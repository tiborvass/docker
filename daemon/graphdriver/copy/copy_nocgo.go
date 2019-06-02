// +build linux,!cgo

package copy // import "github.com/tiborvass/docker/daemon/graphdriver/copy"

import (
	"os"

	"golang.org/x/sys/unix"
)

func fiClone(srcFile, dstFile *os.File) error {
	return unix.ENOSYS
}
