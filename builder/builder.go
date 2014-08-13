package builder

import (
	"github.com/tiborvass/docker/builder/evaluator"
	"github.com/tiborvass/docker/runconfig"
)

// Create a new builder.
func NewBuilder(opts *evaluator.BuildOpts) *evaluator.BuildFile {
	return &evaluator.BuildFile{
		Dockerfile:    nil,
		Config:        &runconfig.Config{},
		Options:       opts,
		TmpContainers: evaluator.UniqueMap{},
		TmpImages:     evaluator.UniqueMap{},
	}
}
