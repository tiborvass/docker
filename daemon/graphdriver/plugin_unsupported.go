// +build !experimental

package graphdriver

import "github.com/tiborvass/docker/pkg/plugingetter"

func lookupPlugin(name, home string, opts []string, pg plugingetter.PluginGetter) (Driver, error) {
	return nil, ErrNotSupported
}
