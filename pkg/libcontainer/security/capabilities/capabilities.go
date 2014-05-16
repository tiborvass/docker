package capabilities

import (
	"os"

	"github.com/dotcloud/docker/pkg/libcontainer"
	"github.com/syndtr/gocapability/capability"
)

const allCapabilityTypes = capability.CAPS | capability.BOUNDS

// DropCapabilities drops all capabilities for the current process expect those specified in the container configuration.
func DropCapabilities(container *libcontainer.Container) error {
	c, err := capability.NewPid(os.Getpid())
	if err != nil {
		return err
	}

	keep := getEnabledCapabilities(container)
	c.Clear(allCapabilityTypes)
	c.Set(allCapabilityTypes, keep...)

	if err := c.Apply(allCapabilityTypes); err != nil {
		return err
	}
	return nil
}

// getCapabilitiesMask returns the capabilities that should not be dropped by the container.
func getEnabledCapabilities(container *libcontainer.Container) []capability.Cap {
	keep := []capability.Cap{}
	for key, enabled := range container.CapabilitiesMask {
		if enabled {
			if c := libcontainer.GetCapability(key); c != nil {
				keep = append(keep, c.Value)
			}
		}
	}
	return keep
}
