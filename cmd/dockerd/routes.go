// +build !experimental

package main

import "github.com/tiborvass/docker/api/server/router"

func addExperimentalRouters(routers []router.Router) []router.Router {
	return routers
}
