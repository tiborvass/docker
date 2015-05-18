// +build windows

package execdrivers

import (
	"fmt"

	"github.com/tiborvass/docker/daemon/execdriver"
	"github.com/tiborvass/docker/daemon/execdriver/windows"
	"github.com/tiborvass/docker/pkg/sysinfo"
)

func NewDriver(name string, options []string, root, libPath, initPath string, sysInfo *sysinfo.SysInfo) (execdriver.Driver, error) {
	switch name {
	case "windows":
		return windows.NewDriver(root, initPath)
	}
	return nil, fmt.Errorf("unknown exec driver %s", name)
}
