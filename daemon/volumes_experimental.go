// +build experimental

package daemon

import (
	"path/filepath"

	"github.com/tiborvass/docker/runconfig"
	"github.com/tiborvass/docker/volume"
	"github.com/tiborvass/docker/volume/drivers"
)

func getVolumeDriver(name string) (volume.Driver, error) {
	if name == "" {
		name = volume.DefaultDriverName
	}
	return volumedrivers.Lookup(name)
}

func parseVolumeSource(spec string, config *runconfig.Config) (string, string, error) {
	if !filepath.IsAbs(spec) {
		return spec, "", nil
	}

	return "", spec, nil
}
