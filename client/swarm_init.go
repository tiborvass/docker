package client // import "github.com/tiborvass/docker/client"

import (
	"context"
	"encoding/json"

	"github.com/tiborvass/docker/api/types/swarm"
)

// SwarmInit initializes the swarm.
func (cli *Client) SwarmInit(ctx context.Context, req swarm.InitRequest) (string, error) {
	serverResp, err := cli.post(ctx, "/swarm/init", nil, req, nil)
	if err != nil {
		return "", err
	}

	var response string
	err = json.NewDecoder(serverResp.body).Decode(&response)
	ensureReaderClosed(serverResp)
	return response, err
}
