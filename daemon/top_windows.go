package daemon

import (
	derr "github.com/tiborvass/docker/api/errors"
	"github.com/tiborvass/docker/api/types"
)

// ContainerTop is not supported on Windows and returns an error.
func (daemon *Daemon) ContainerTop(name string, psArgs string) (*types.ContainerProcessList, error) {
	return nil, derr.ErrorCodeNoTop
}
