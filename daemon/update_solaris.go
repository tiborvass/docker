package daemon

import (
	"github.com/tiborvass/docker/libcontainerd"
	"github.com/docker/engine-api/types/container"
)

func toContainerdResources(resources container.Resources) libcontainerd.Resources {
	var r libcontainerd.Resources
	return r
}
