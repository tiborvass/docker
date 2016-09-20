// +build experimental

package graphdriver

import (
	"fmt"
	"io"

	"github.com/tiborvass/docker/plugin/getter"
)

type pluginClient interface {
	// Call calls the specified method with the specified arguments for the plugin.
	Call(string, interface{}, interface{}) error
	// Stream calls the specified method with the specified arguments for the plugin and returns the response IO stream
	Stream(string, interface{}) (io.ReadCloser, error)
	// SendFile calls the specified method, and passes through the IO stream
	SendFile(string, io.Reader, interface{}) error
}

func lookupPlugin(name, home string, opts []string, pluginGetter getter.PluginGetter) (Driver, error) {
	pl, err := pluginGetter.Get(name, "GraphDriver", getter.LOOKUP)
	if err != nil {
		return nil, fmt.Errorf("Error looking up graphdriver plugin %s: %v", name, err)
	}
	return newPluginDriver(name, home, opts, pl.Client())
}

func newPluginDriver(name, home string, opts []string, c pluginClient) (Driver, error) {
	proxy := &graphDriverProxy{name, c}
	return proxy, proxy.Init(home, opts)
}
