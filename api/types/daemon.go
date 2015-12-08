package types

import "github.com/docker/docker/runconfig"

// ContainerCreateConfig is the parameter set to ContainerCreate()
type ContainerCreateConfig struct {
	Name            string
	Config          *runconfig.Config
	HostConfig      *runconfig.HostConfig
	AdjustCPUShares bool
}
