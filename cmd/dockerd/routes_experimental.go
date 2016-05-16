// +build experimental

package main

import (
	"github.com/tiborvass/docker/api/server/router"
	pluginrouter "github.com/tiborvass/docker/api/server/router/plugin"
	"github.com/tiborvass/docker/plugin"
)

func addExperimentalRouters(routers []router.Router) []router.Router {
	return append(routers, pluginrouter.NewRouter(plugin.GetManager()))
}
