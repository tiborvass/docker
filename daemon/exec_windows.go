package daemon

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/daemon/execdriver"
)

// setPlatformSpecificExecProcessConfig sets platform-specific fields in the
// ProcessConfig structure. This is a no-op on Windows
func setPlatformSpecificExecProcessConfig(config *types.ExecConfig, container *container.Container, pc *execdriver.ProcessConfig) {
}
