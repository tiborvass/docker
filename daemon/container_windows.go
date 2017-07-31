//+build windows

package daemon

import (
	"github.com/moby/moby-core/container"
)

func (daemon *Daemon) saveApparmorConfig(container *container.Container) error {
	return nil
}
