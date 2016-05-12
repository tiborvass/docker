// +build !experimental

package main

import (
	"github.com/tiborvass/docker/api/server/httputils"
	"github.com/tiborvass/docker/api/server/router"
	"github.com/tiborvass/docker/daemon"
)

func addExperimentalRouters(routers []router.Router, d *daemon.Daemon, decoder httputils.ContainerDecoder) []router.Router {
	return routers
}
