package secret

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/filters"
	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/tiborvass/docker/client"
	"golang.org/x/net/context"
)

func getSecretsByName(ctx context.Context, client client.APIClient, names []string) ([]swarm.Secret, error) {
	args := filters.NewArgs()
	for _, n := range names {
		args.Add("names", n)
	}

	return client.SecretList(ctx, types.SecretListOptions{
		Filters: args,
	})
}
