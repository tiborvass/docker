package daemon

import (
	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api/types"
	containertypes "github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/container"
	derr "github.com/tiborvass/docker/errors"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/layer"
	"github.com/tiborvass/docker/pkg/idtools"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/tiborvass/docker/volume"
	"github.com/opencontainers/runc/libcontainer/label"
)

// ContainerCreate creates a container.
func (daemon *Daemon) ContainerCreate(params types.ContainerCreateConfig) (types.ContainerCreateResponse, error) {
	if params.Config == nil {
		return types.ContainerCreateResponse{}, derr.ErrorCodeEmptyConfig
	}

	warnings, err := daemon.verifyContainerSettings(params.HostConfig, params.Config)
	if err != nil {
		return types.ContainerCreateResponse{Warnings: warnings}, err
	}

	if params.HostConfig == nil {
		params.HostConfig = &containertypes.HostConfig{}
	}
	err = daemon.adaptContainerSettings(params.HostConfig, params.AdjustCPUShares)
	if err != nil {
		return types.ContainerCreateResponse{Warnings: warnings}, err
	}

	container, err := daemon.create(params)
	if err != nil {
		return types.ContainerCreateResponse{Warnings: warnings}, daemon.imageNotExistToErrcode(err)
	}

	return types.ContainerCreateResponse{ID: container.ID, Warnings: warnings}, nil
}

// Create creates a new container from the given configuration with a given name.
func (daemon *Daemon) create(params types.ContainerCreateConfig) (retC *container.Container, retErr error) {
	var (
		container *container.Container
		img       *image.Image
		imgID     image.ID
		err       error
	)

	if params.Config.Image != "" {
		img, err = daemon.GetImage(params.Config.Image)
		if err != nil {
			return nil, err
		}
		imgID = img.ID()
	}

	if err := daemon.mergeAndVerifyConfig(params.Config, img); err != nil {
		return nil, err
	}

	if container, err = daemon.newContainer(params.Name, params.Config, imgID); err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			if err := daemon.ContainerRm(container.ID, &types.ContainerRmConfig{ForceRemove: true}); err != nil {
				logrus.Errorf("Clean up Error! Cannot destroy container %s: %v", container.ID, err)
			}
		}
	}()

	if err := daemon.setSecurityOptions(container, params.HostConfig); err != nil {
		return nil, err
	}

	// Set RWLayer for container after mount labels have been set
	if err := daemon.setRWLayer(container); err != nil {
		return nil, err
	}

	if err := daemon.Register(container); err != nil {
		return nil, err
	}
	rootUID, rootGID, err := idtools.GetRootUIDGID(daemon.uidMaps, daemon.gidMaps)
	if err != nil {
		return nil, err
	}
	if err := idtools.MkdirAs(container.Root, 0700, rootUID, rootGID); err != nil {
		return nil, err
	}

	if err := daemon.setHostConfig(container, params.HostConfig); err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			if err := daemon.removeMountPoints(container, true); err != nil {
				logrus.Error(err)
			}
		}
	}()

	if err := daemon.createContainerPlatformSpecificSettings(container, params.Config, params.HostConfig, img); err != nil {
		return nil, err
	}

	if err := daemon.updateContainerNetworkSettings(container); err != nil {
		return nil, err
	}

	if err := container.ToDiskLocking(); err != nil {
		logrus.Errorf("Error saving new container to disk: %v", err)
		return nil, err
	}
	daemon.LogContainerEvent(container, "create")
	return container, nil
}

func (daemon *Daemon) generateSecurityOpt(ipcMode containertypes.IpcMode, pidMode containertypes.PidMode) ([]string, error) {
	if ipcMode.IsHost() || pidMode.IsHost() {
		return label.DisableSecOpt(), nil
	}
	if ipcContainer := ipcMode.Container(); ipcContainer != "" {
		c, err := daemon.GetContainer(ipcContainer)
		if err != nil {
			return nil, err
		}

		return label.DupSecOpt(c.ProcessLabel), nil
	}
	return nil, nil
}

func (daemon *Daemon) setRWLayer(container *container.Container) error {
	var layerID layer.ChainID
	if container.ImageID != "" {
		img, err := daemon.imageStore.Get(container.ImageID)
		if err != nil {
			return err
		}
		layerID = img.RootFS.ChainID()
	}
	rwLayer, err := daemon.layerStore.CreateRWLayer(container.ID, layerID, container.MountLabel, daemon.setupInitLayer)
	if err != nil {
		return err
	}
	container.RWLayer = rwLayer

	return nil
}

// VolumeCreate creates a volume with the specified name, driver, and opts
// This is called directly from the remote API
func (daemon *Daemon) VolumeCreate(name, driverName string, opts map[string]string) (*types.Volume, error) {
	if name == "" {
		name = stringid.GenerateNonCryptoID()
	}

	v, err := daemon.volumes.Create(name, driverName, opts)
	if err != nil {
		return nil, err
	}

	// keep "docker run -v existing_volume:/foo --volume-driver other_driver" work
	if (driverName != "" && v.DriverName() != driverName) || (driverName == "" && v.DriverName() != volume.DefaultDriverName) {
		return nil, derr.ErrorVolumeNameTaken.WithArgs(name, v.DriverName())
	}

	if driverName == "" {
		driverName = volume.DefaultDriverName
	}
	daemon.LogVolumeEvent(name, "create", map[string]string{"driver": driverName})
	return volumeToAPIType(v), nil
}
