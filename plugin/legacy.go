// +build !experimental

package plugin

import "github.com/docker/docker/pkg/plugins"

// FindWithCapabilities returns a list of plugins matching all given capabilities.
func FindWithCapabilities(capabilities ...string) ([]Plugin, error) {
	pl, err := plugins.GetAll(capabilities...)
	if err != nil {
		return nil, err
	}
	result := make([]Plugin, len(pl))
	for i, p := range pl {
		result[i] = p
	}
	return result, nil
}

// LookupWithCapability returns a plugin matching the given name and capability.
func LookupWithCapability(name, capability string) (Plugin, error) {
	return plugins.Get(name, capability)
}
