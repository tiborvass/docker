package daemon

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/daemon/exec"
	"github.com/tiborvass/docker/libcontainerd"
)

func execSetPlatformOpt(c *container.Container, ec *exec.Config, p *libcontainerd.Process) error {
	return nil
}
