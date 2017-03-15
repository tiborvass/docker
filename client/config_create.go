package client

import (
	"encoding/json"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/swarm"
	"golang.org/x/net/context"
)

// ConfigCreate creates a new Config.
func (cli *Client) ConfigCreate(ctx context.Context, config swarm.ConfigSpec) (types.ConfigCreateResponse, error) {
	var response types.ConfigCreateResponse
	resp, err := cli.post(ctx, "/configs/create", nil, config, nil)
	if err != nil {
		return response, err
	}

	err = json.NewDecoder(resp.body).Decode(&response)
	ensureReaderClosed(resp)
	return response, err
}
