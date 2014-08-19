package builder

import (
	"github.com/tiborvass/docker/runconfig"
)

// Create a new builder. See
func NewBuilder(opts *BuildOpts) *BuildFile {
	return &BuildFile{
		Dockerfile:    nil,
		Config:        &runconfig.Config{},
		Options:       opts,
		TmpContainers: map[string]struct{}{},
	}
}
