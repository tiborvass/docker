package server

import (
	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api"
	"github.com/tiborvass/docker/api/server/httputils"
	"github.com/tiborvass/docker/api/server/middleware"
	"github.com/tiborvass/docker/dockerversion"
	"github.com/tiborvass/docker/pkg/authorization"
)

// handleWithGlobalMiddlwares wraps the handler function for a request with
// the server's global middlewares. The order of the middlewares is backwards,
// meaning that the first in the list will be evaluated last.
func (s *Server) handleWithGlobalMiddlewares(handler httputils.APIFunc) httputils.APIFunc {
	next := handler

	handleVersion := middleware.NewVersionMiddleware(dockerversion.Version, api.DefaultVersion, api.MinVersion)
	next = handleVersion(next)

	if s.cfg.EnableCors {
		handleCORS := middleware.NewCORSMiddleware(s.cfg.CorsHeaders)
		next = handleCORS(next)
	}

	handleUserAgent := middleware.NewUserAgentMiddleware(s.cfg.Version)
	next = handleUserAgent(next)

	// Only want this on debug level
	if s.cfg.Logging && logrus.GetLevel() == logrus.DebugLevel {
		next = middleware.DebugRequestMiddleware(next)
	}

	if len(s.cfg.AuthorizationPluginNames) > 0 {
		s.authZPlugins = authorization.NewPlugins(s.cfg.AuthorizationPluginNames)
		handleAuthorization := middleware.NewAuthorizationMiddleware(s.authZPlugins)
		next = handleAuthorization(next)
	}

	return next
}
