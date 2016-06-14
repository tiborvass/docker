package daemon

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/libcontainerd"
)

func (daemon *Daemon) getLibcontainerdCreateOptions(container *container.Container) (*[]libcontainerd.CreateOption, error) {
	return &[]libcontainerd.CreateOption{}, nil
}
