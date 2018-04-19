package client // import "github.com/tiborvass/docker/client"

import (
	"context"
	"encoding/json"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/swarm"
)

// ConfigCreate creates a new Config.
func (cli *Client) ConfigCreate(ctx context.Context, config swarm.ConfigSpec) (types.ConfigCreateResponse, error) {
	var response types.ConfigCreateResponse
	if err := cli.NewVersionError("1.30", "config create"); err != nil {
		return response, err
	}
	resp, err := cli.post(ctx, "/configs/create", nil, config, nil)
	if err != nil {
		return response, err
	}

	err = json.NewDecoder(resp.body).Decode(&response)
	ensureReaderClosed(resp)
	return response, err
}
