// +build !experimental

package store

import (
	"github.com/tiborvass/docker/pkg/plugins"
)

// FindWithCapability returns a list of plugins matching the given capability.
func FindWithCapability(capability string) ([]CompatPlugin, error) {
	pl, err := plugins.GetAll(capability)
	if err != nil {
		return nil, err
	}
	result := make([]CompatPlugin, len(pl))
	for i, p := range pl {
		result[i] = p
	}
	return result, nil
}

// LookupWithCapability returns a plugin matching the given name and capability.
func LookupWithCapability(name, capability string, _ int) (CompatPlugin, error) {
	return plugins.Get(name, capability)
}
