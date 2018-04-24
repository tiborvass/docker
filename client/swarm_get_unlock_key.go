package client // import "github.com/tiborvass/docker/client"

import (
	"context"
	"encoding/json"

	"github.com/tiborvass/docker/api/types"
)

// SwarmGetUnlockKey retrieves the swarm's unlock key.
func (cli *Client) SwarmGetUnlockKey(ctx context.Context) (types.SwarmUnlockKeyResponse, error) {
	serverResp, err := cli.get(ctx, "/swarm/unlockkey", nil, nil)
	if err != nil {
		return types.SwarmUnlockKeyResponse{}, err
	}

	var response types.SwarmUnlockKeyResponse
	err = json.NewDecoder(serverResp.body).Decode(&response)
	ensureReaderClosed(serverResp)
	return response, err
}
