package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/backend"
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/daemon/exec"
)

// This sets platform-specific fields
func setPlatformSpecificContainerFields(container *container.Container, contJSONBase *types.ContainerJSONBase) *types.ContainerJSONBase {
	return contJSONBase
}

// containerInspectPre120 get containers for pre 1.20 APIs.
func (daemon *Daemon) containerInspectPre120(name string) (*types.ContainerJSON, error) {
	return daemon.ContainerInspectCurrent(name, false)
}

func inspectExecProcessConfig(e *exec.Config) *backend.ExecProcessConfig {
	return &backend.ExecProcessConfig{
		Tty:        e.Tty,
		Entrypoint: e.Entrypoint,
		Arguments:  e.Args,
	}
}
