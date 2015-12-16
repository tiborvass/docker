package builder

import "github.com/tiborvass/docker/runconfig"

// Image represents a Docker image used by the builder.
type Image interface {
	ID() string
	Config() *runconfig.Config
}
