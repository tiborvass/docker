package volume

import (
	// TODO return types need to be refactored into pkg
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/filters"
)

// Backend is the methods that need to be implemented to provide
// volume specific functionality
type Backend interface {
	Volumes(filter string) ([]*types.Volume, []string, error)
	VolumeInspect(name string) (*types.Volume, error)
	VolumeCreate(name, driverName string, opts, labels map[string]string) (*types.Volume, error)
	VolumeRm(name string, force bool) error
	VolumesPrune(pruneFilters filters.Args) (*types.VolumesPruneReport, error)
}
