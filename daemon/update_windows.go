package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/libcontainerd"
)

func toContainerdResources(resources container.Resources) *libcontainerd.Resources {
	// We don't support update, so do nothing
	return nil
}
