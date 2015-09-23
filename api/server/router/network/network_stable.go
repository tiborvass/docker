// +build !experimental

package network

import (
	"net/http"

	"github.com/tiborvass/docker/api/server/router"
	"github.com/tiborvass/docker/daemon"
	"github.com/gorilla/mux"
)

// NewRouter initializes a new network router
func NewRouter(d *daemon.Daemon) router.Router {
	return networkRouter{}
}

// Register adds the filtered handler to the mux.
func (n networkRoute) Register(m *mux.Router, handler http.Handler) {
}
