package daemon

import (
	"github.com/tiborvass/docker/runconfig"
)

// createContainerPlatformSpecificSettings performs platform specific container create functionality
func createContainerPlatformSpecificSettings(container *Container, config *runconfig.Config) error {
	return nil
}
