package daemon

import (
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/runconfig"
)

// createContainerPlatformSpecificSettings performs platform specific container create functionality
func createContainerPlatformSpecificSettings(container *Container, config *runconfig.Config, hostConfig *runconfig.HostConfig, img *image.Image) error {
	return nil
}
