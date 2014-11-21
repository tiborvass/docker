package server

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/engine"
	"github.com/docker/docker/extensions"
)

func postCmd(eng *engine.Engine, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	eng.ServeHTTP(w, r)
}
