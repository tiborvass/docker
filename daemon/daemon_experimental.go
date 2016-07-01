// +build experimental

package daemon

import (
	"github.com/tiborvass/docker/plugin"
	"github.com/docker/engine-api/types/container"
)

func (daemon *Daemon) verifyExperimentalContainerSettings(hostConfig *container.HostConfig, config *container.Config) ([]string, error) {
	return nil, nil
}

func pluginShutdown() {
	plugin.GetManager().Shutdown()
}
