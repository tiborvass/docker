package daemon

import (
	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/autoversion"
)

// AuthenticateToRegistry checks the validity of credentials in authConfig
func (daemon *Daemon) AuthenticateToRegistry(ctx context.Context, authConfig *types.AuthConfig) (string, string, error) {
	return daemon.RegistryService.Auth(ctx, authConfig, autoversion.DockerUserAgent(ctx))
}
