package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/pkg/sysinfo"
)

// fillPlatformInfo fills the platform related info.
func (daemon *Daemon) fillPlatformInfo(v *types.Info, sysInfo *sysinfo.SysInfo) {
}

func fillDriverWarnings(v *types.Info) {
}
