// +build !windows

package daemon

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/versions/v1p19"
)

// This sets platform-specific fields
func setPlatformSpecificContainerFields(container *Container, contJSONBase *types.ContainerJSONBase) *types.ContainerJSONBase {
	contJSONBase.AppArmorProfile = container.AppArmorProfile
	contJSONBase.ResolvConfPath = container.ResolvConfPath
	contJSONBase.HostnamePath = container.HostnamePath
	contJSONBase.HostsPath = container.HostsPath

	return contJSONBase
}

// ContainerInspectPre120 gets containers for pre 1.20 APIs.
func (daemon *Daemon) ContainerInspectPre120(name string) (*v1p19.ContainerJSON, error) {
	container, err := daemon.Get(name)
	if err != nil {
		return nil, err
	}

	container.Lock()
	defer container.Unlock()

	base, err := daemon.getInspectData(container, false)
	if err != nil {
		return nil, err
	}

	volumes := make(map[string]string)
	volumesRW := make(map[string]bool)
	for _, m := range container.MountPoints {
		volumes[m.Destination] = m.Path()
		volumesRW[m.Destination] = m.RW
	}

	config := &v1p19.ContainerConfig{
		Config:          container.Config,
		MacAddress:      container.Config.MacAddress,
		NetworkDisabled: container.Config.NetworkDisabled,
		ExposedPorts:    container.Config.ExposedPorts,
		VolumeDriver:    container.hostConfig.VolumeDriver,
		Memory:          container.hostConfig.Memory,
		MemorySwap:      container.hostConfig.MemorySwap,
		CPUShares:       container.hostConfig.CPUShares,
		CPUSet:          container.hostConfig.CpusetCpus,
	}
	networkSettings := daemon.getBackwardsCompatibleNetworkSettings(container.NetworkSettings)

	return &v1p19.ContainerJSON{
		ContainerJSONBase: base,
		Volumes:           volumes,
		VolumesRW:         volumesRW,
		Config:            config,
		NetworkSettings:   networkSettings,
	}, nil
}

func addMountPoints(container *Container) []types.MountPoint {
	mountPoints := make([]types.MountPoint, 0, len(container.MountPoints))
	for _, m := range container.MountPoints {
		mountPoints = append(mountPoints, types.MountPoint{
			Name:        m.Name,
			Source:      m.Path(),
			Destination: m.Destination,
			Driver:      m.Driver,
			Mode:        m.Mode,
			RW:          m.RW,
		})
	}
	return mountPoints
}
