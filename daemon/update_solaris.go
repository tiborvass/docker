package daemon

import (
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/libcontainerd"
)

func toContainerdResources(resources container.Resources) libcontainerd.Resources {
	var r libcontainerd.Resources
	return r
}
