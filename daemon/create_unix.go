// +build !windows

package daemon

import (
	"os"
	"path/filepath"

	containertypes "github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/container"
	derr "github.com/tiborvass/docker/errors"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/tiborvass/docker/volume"
	"github.com/opencontainers/runc/libcontainer/label"
)

// createContainerPlatformSpecificSettings performs platform specific container create functionality
func (daemon *Daemon) createContainerPlatformSpecificSettings(container *container.Container, config *containertypes.Config, hostConfig *containertypes.HostConfig, img *image.Image) error {
	if err := daemon.Mount(container); err != nil {
		return err
	}
	defer daemon.Unmount(container)

	for spec := range config.Volumes {
		name := stringid.GenerateNonCryptoID()
		destination := filepath.Clean(spec)

		// Skip volumes for which we already have something mounted on that
		// destination because of a --volume-from.
		if container.IsDestinationMounted(destination) {
			continue
		}
		path, err := container.GetResourcePath(destination)
		if err != nil {
			return err
		}

		stat, err := os.Stat(path)
		if err == nil && !stat.IsDir() {
			return derr.ErrorCodeMountOverFile.WithArgs(path)
		}

		volumeDriver := hostConfig.VolumeDriver
		if destination != "" && img != nil {
			if _, ok := img.ContainerConfig.Volumes[destination]; ok {
				// check for whether bind is not specified and then set to local
				if _, ok := container.MountPoints[destination]; !ok {
					volumeDriver = volume.DefaultDriverName
				}
			}
		}

		v, err := daemon.volumes.CreateWithRef(name, volumeDriver, container.ID, nil)
		if err != nil {
			return err
		}

		if err := label.Relabel(v.Path(), container.MountLabel, true); err != nil {
			return err
		}

		// never attempt to copy existing content in a container FS to a shared volume
		if v.DriverName() == volume.DefaultDriverName {
			if err := container.CopyImagePathContent(v, destination); err != nil {
				return err
			}
		}

		container.AddMountPointWithVolume(destination, v, true)
	}
	return nil
}
