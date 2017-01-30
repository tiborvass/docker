//+build windows

package daemon

import (
	"github.com/tiborvass/docker/container"
)

func (daemon *Daemon) saveApparmorConfig(container *container.Container) error {
	return nil
}
