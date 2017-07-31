package daemon

import (
	"github.com/moby/moby-core/container"
	"github.com/moby/moby-core/daemon/exec"
	"github.com/moby/moby-core/libcontainerd"
)

func execSetPlatformOpt(c *container.Container, ec *exec.Config, p *libcontainerd.Process) error {
	return nil
}
