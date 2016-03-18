package daemon

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/daemon/caps"
	"github.com/tiborvass/docker/daemon/exec"
	"github.com/tiborvass/docker/libcontainerd"
)

func execSetPlatformOpt(c *container.Container, ec *exec.Config, p *libcontainerd.Process) error {
	if len(ec.User) > 0 {
		uid, gid, additionalGids, err := getUser(c, ec.User)
		if err != nil {
			return err
		}
		p.User = &libcontainerd.User{
			UID:            uid,
			GID:            gid,
			AdditionalGids: additionalGids,
		}
	}
	if ec.Privileged {
		p.Capabilities = caps.GetAllCapabilities()
	}
	return nil
}
