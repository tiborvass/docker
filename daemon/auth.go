package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"context"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/dockerversion"
)

// AuthenticateToRegistry checks the validity of credentials in authConfig
func (daemon *Daemon) AuthenticateToRegistry(ctx context.Context, authConfig *types.AuthConfig) (string, string, error) {
	return daemon.RegistryService.Auth(ctx, authConfig, dockerversion.DockerUserAgent(ctx))
}
