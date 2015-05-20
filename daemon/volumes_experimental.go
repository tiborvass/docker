// +build experimental

package daemon

import (
	"github.com/tiborvass/docker/volume"
	"github.com/tiborvass/docker/volume/drivers"
)

func getVolumeDriver(name string) (volume.Driver, error) {
	if name == "" {
		name = volume.DefaultDriverName
	}
	return volumedrivers.Lookup(name)
}
