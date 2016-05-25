package daemon

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/libcontainerd"
	"github.com/tiborvass/docker/oci"
)

func (daemon *Daemon) createSpec(c *container.Container) (*libcontainerd.Spec, error) {
	s := oci.DefaultSpec()
	return (*libcontainerd.Spec)(&s), nil
}
