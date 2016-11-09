package daemon

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/daemon/exec"
	"github.com/tiborvass/docker/libcontainerd"
)

func execSetPlatformOpt(c *container.Container, ec *exec.Config, p *libcontainerd.Process) error {
	// Process arguments need to be escaped before sending to OCI.
	p.Args = escapeArgs(p.Args)
	p.User.Username = ec.User
	return nil
}
