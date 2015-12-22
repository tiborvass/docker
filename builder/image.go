package builder

import "github.com/tiborvass/docker/api/types/container"

// Image represents a Docker image used by the builder.
type Image interface {
	ID() string
	Config() *container.Config
}
