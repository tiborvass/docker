// +build !windows

package client

import (
	"os"
	"syscall"
)

func checkSigChld(s os.Signal) bool {
	return s == syscall.SIGCHLD
}
