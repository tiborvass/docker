package daemon

import (
	derr "github.com/tiborvass/docker/errors"
)

// ContainerUnpause unpauses a container
func (daemon *Daemon) ContainerUnpause(name string) error {
	container, err := daemon.Get(name)
	if err != nil {
		return err
	}

	if err := container.unpause(); err != nil {
		return derr.ErrorCodeCantUnpause.WithArgs(name, err)
	}

	return nil
}
