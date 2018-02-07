package session // import "github.com/tiborvass/docker/api/server/router/session"

import (
	"net/http"

	"github.com/tiborvass/docker/errdefs"
	"golang.org/x/net/context"
)

func (sr *sessionRouter) startSession(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	err := sr.backend.HandleHTTPRequest(ctx, w, r)
	if err != nil {
		return errdefs.InvalidParameter(err)
	}
	return nil
}
