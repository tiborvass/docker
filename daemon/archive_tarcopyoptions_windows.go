// +build windows

package daemon

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/pkg/archive"
)

func (daemon *Daemon) tarCopyOptions(container *container.Container, noOverwriteDirNonDir bool) (*archive.TarOptions, error) {
	return daemon.defaultTarCopyOptions(noOverwriteDirNonDir), nil
}
