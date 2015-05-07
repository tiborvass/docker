// +build linux

package execdrivers

import (
	"fmt"
	"path"

	"github.com/tiborvass/docker/daemon/execdriver"
	"github.com/tiborvass/docker/daemon/execdriver/lxc"
	"github.com/tiborvass/docker/daemon/execdriver/native"
	"github.com/tiborvass/docker/pkg/sysinfo"
)

func NewDriver(name string, options []string, root, libPath, initPath string, sysInfo *sysinfo.SysInfo) (execdriver.Driver, error) {
	switch name {
	case "lxc":
		// we want to give the lxc driver the full docker root because it needs
		// to access and write config and template files in /var/lib/docker/containers/*
		// to be backwards compatible
		return lxc.NewDriver(root, libPath, initPath, sysInfo.AppArmor)
	case "native":
		return native.NewDriver(path.Join(root, "execdriver", "native"), initPath, options)
	}
	return nil, fmt.Errorf("unknown exec driver %s", name)
}
