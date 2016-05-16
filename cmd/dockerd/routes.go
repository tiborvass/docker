// +build !experimental

package main

import (
	"github.com/docker/docker/api/server/router"
	"github.com/docker/docker/api/server/router/build"
	"github.com/docker/docker/api/server/router/container"
	"github.com/docker/docker/api/server/router/image"
	systemrouter "github.com/docker/docker/api/server/router/system"
	"github.com/docker/docker/api/server/router/volume"
	"github.com/docker/docker/builder/dockerfile"
	"github.com/docker/docker/daemon"
	"github.com/docker/docker/runconfig"
)

func routes(d *daemon.Daemon) []router.Router {
	decoder := runconfig.ContainerDecoder{}

	routers := []router.Router{
		container.NewRouter(d, decoder),
		image.NewRouter(d, decoder),
		systemrouter.NewRouter(d),
		volume.NewRouter(d),
		build.NewRouter(dockerfile.NewBuildManager(d)),
	}
	return routers
}
