// +build linux

package execdrivers

import (
	"path"

	"github.com/tiborvass/docker/daemon/execdriver"
	"github.com/tiborvass/docker/daemon/execdriver/native"
	"github.com/tiborvass/docker/pkg/sysinfo"
)

// NewDriver returns a new execdriver.Driver from the given name configured with the provided options.
func NewDriver(options []string, root, libPath string, sysInfo *sysinfo.SysInfo) (execdriver.Driver, error) {
	return native.NewDriver(path.Join(root, "execdriver", "native"), options)
}
