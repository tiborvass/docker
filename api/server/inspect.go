package server

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"github.com/tiborvass/docker/pkg/version"
)

// getContainersByName inspects containers configuration and serializes it as json.
func (s *Server) getContainersByName(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	var json interface{}
	var err error

	version, _ := ctx.Value("api-version").(version.Version)

	switch {
	case version.LessThan("1.20"):
		json, err = s.daemon.ContainerInspectPre120(vars["name"])
	case version.Equal("1.20"):
		json, err = s.daemon.ContainerInspect120(vars["name"])
	default:
		json, err = s.daemon.ContainerInspect(vars["name"])
	}

	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, json)
}
