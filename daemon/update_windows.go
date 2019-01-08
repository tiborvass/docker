package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"github.com/tiborvass/docker/api/types/container"
	libcontainerdtypes "github.com/tiborvass/docker/libcontainerd/types"
)

func toContainerdResources(resources container.Resources) *libcontainerdtypes.Resources {
	// We don't support update, so do nothing
	return nil
}
