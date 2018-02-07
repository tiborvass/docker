// +build !linux,!darwin,!freebsd,!windows

package signal // import "github.com/tiborvass/docker/pkg/signal"

import (
	"syscall"
)

// SignalMap is an empty map of signals for unsupported platform.
var SignalMap = map[string]syscall.Signal{}
