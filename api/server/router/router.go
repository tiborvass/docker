package router

import "github.com/tiborvass/docker/api/server/httputils"

// Router defines an interface to specify a group of routes to add the the docker server.
type Router interface {
	Routes() []Route
}

// Route defines an individual API route in the docker server.
type Route interface {
	// Handler returns the raw function to create the http handler.
	Handler() httputils.APIFunc
	// Method returns the http method that the route responds to.
	Method() string
	// Path returns the subpath where the route responds to.
	Path() string
}
