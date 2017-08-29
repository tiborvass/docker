package config

import (
	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/tiborvass/docker/daemon/cluster/convert"
	"github.com/docker/swarmkit/api/genericresource"
)

// ParseGenericResources parses and validates the specified string as a list of GenericResource
func ParseGenericResources(value string) ([]swarm.GenericResource, error) {
	if value == "" {
		return nil, nil
	}

	resources, err := genericresource.Parse(value)
	if err != nil {
		return nil, err
	}

	obj := convert.GenericResourcesFromGRPC(resources)
	return obj, nil
}
