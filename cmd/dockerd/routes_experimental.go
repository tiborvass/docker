// +build experimental

package main

import (
	"github.com/tiborvass/docker/api/server/httputils"
	"github.com/tiborvass/docker/api/server/router"
	checkpointrouter "github.com/tiborvass/docker/api/server/router/checkpoint"
	pluginrouter "github.com/tiborvass/docker/api/server/router/plugin"
	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/plugin"
)

func addExperimentalRouters(routers []router.Router, d *daemon.Daemon, decoder httputils.ContainerDecoder) []router.Router {
	return append(routers, checkpointrouter.NewRouter(d, decoder), pluginrouter.NewRouter(plugin.GetManager()))
}
