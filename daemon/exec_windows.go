package daemon

import (
	"github.com/moby/moby-core/container"
	"github.com/moby/moby-core/daemon/exec"
	"github.com/moby/moby-core/libcontainerd"
)

func execSetPlatformOpt(c *container.Container, ec *exec.Config, p *libcontainerd.Process) error {
	// Process arguments need to be escaped before sending to OCI.
	if c.Platform == "windows" {
		p.Args = escapeArgs(p.Args)
		p.User.Username = ec.User
	}
	return nil
}
