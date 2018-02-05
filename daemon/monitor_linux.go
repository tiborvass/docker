package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/libcontainerd"
)

// postRunProcessing perfoms any processing needed on the container after it has stopped.
func (daemon *Daemon) postRunProcessing(_ *container.Container, _ libcontainerd.EventInfo) error {
	return nil
}
