package daemon

import (
	"github.com/tiborvass/docker/api/types"
	derr "github.com/tiborvass/docker/errors"
)

// ContainerTop is not supported on Windows and returns an error.
func (daemon *Daemon) ContainerTop(name string, psArgs string) (*types.ContainerProcessList, error) {
	return nil, derr.ErrorCodeNoTop
}
