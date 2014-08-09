// +build !windows

package client

import (
	"os"
	"syscall"
)

func notifyWinch(sigchan chan os.Signal) {
	gosignal.Notify(sigchan, syscall.SIGWINCH)
}
