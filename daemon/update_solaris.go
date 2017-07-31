package daemon

import (
	"github.com/moby/moby-core/api/types/container"
	"github.com/moby/moby-core/libcontainerd"
)

func toContainerdResources(resources container.Resources) libcontainerd.Resources {
	var r libcontainerd.Resources
	return r
}
