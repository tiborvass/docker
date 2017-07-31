// +build windows

package daemon

import (
	"github.com/moby/moby-core/container"
	"github.com/moby/moby-core/pkg/archive"
)

func (daemon *Daemon) tarCopyOptions(container *container.Container, noOverwriteDirNonDir bool) (*archive.TarOptions, error) {
	return daemon.defaultTarCopyOptions(noOverwriteDirNonDir), nil
}
