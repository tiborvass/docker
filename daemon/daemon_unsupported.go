// +build !linux,!freebsd,!windows

package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"github.com/tiborvass/docker/daemon/config"
	"github.com/tiborvass/docker/pkg/sysinfo"
)

const platformSupported = false

func setupResolvConf(config *config.Config) {
}

// RawSysInfo returns *sysinfo.SysInfo .
func (daemon *Daemon) RawSysInfo(quiet bool) *sysinfo.SysInfo {
	return sysinfo.New(quiet)
}
