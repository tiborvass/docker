package daemon

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/daemon/execdriver"
	"github.com/tiborvass/docker/runconfig"
)

// setPlatformSpecificExecProcessConfig sets platform-specific fields in the
// ProcessConfig structure. This is a no-op on Windows
func setPlatformSpecificExecProcessConfig(config *runconfig.ExecConfig, container *container.Container, pc *execdriver.ProcessConfig) {
}
