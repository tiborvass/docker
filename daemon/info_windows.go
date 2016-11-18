package daemon

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/pkg/sysinfo"
)

// FillPlatformInfo fills the platform related info.
func (daemon *Daemon) FillPlatformInfo(v *types.InfoBase, sysInfo *sysinfo.SysInfo) {
}
