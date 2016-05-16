// +build experimental

package plugin

import (
	"github.com/docker/docker/api/server/router"
)

func (r *pluginRouter) initRoutes() {
	r.routes = []router.Route{
		router.NewGetRoute("/plugins", r.listPlugins),
		router.NewGetRoute("/plugins/{name:.*}", r.inspectPlugin),
		router.NewDeleteRoute("/plugins/{name:.*}", r.removePlugin),
		router.NewPostRoute("/plugins/{name:.*}/enable", r.enablePlugin), // PATCH?
		router.NewPostRoute("/plugins/{name:.*}/disable", r.disablePlugin),
		router.NewPostRoute("/plugins/{name:.*}/install", r.installPlugin), // weird path
		router.NewPostRoute("/plugins/{name:.*}/set", r.setPlugin),
		router.NewPostRoute("/plugins/{name:.*}/push", r.pushPlugin),
	}
}
