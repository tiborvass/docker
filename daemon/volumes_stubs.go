// +build !experimental

package daemon

import (
	"github.com/tiborvass/docker/volume"
	"github.com/tiborvass/docker/volume/drivers"
)

func getVolumeDriver(_ string) (volume.Driver, error) {
	return volumedrivers.Lookup(volume.DefaultDriverName)
}
