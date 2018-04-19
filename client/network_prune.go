package client // import "github.com/tiborvass/docker/client"

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/filters"
)

// NetworksPrune requests the daemon to delete unused networks
func (cli *Client) NetworksPrune(ctx context.Context, pruneFilters filters.Args) (types.NetworksPruneReport, error) {
	var report types.NetworksPruneReport

	if err := cli.NewVersionError("1.25", "network prune"); err != nil {
		return report, err
	}

	query, err := getFiltersQuery(pruneFilters)
	if err != nil {
		return report, err
	}

	serverResp, err := cli.post(ctx, "/networks/prune", query, nil, nil)
	if err != nil {
		return report, err
	}
	defer ensureReaderClosed(serverResp)

	if err := json.NewDecoder(serverResp.body).Decode(&report); err != nil {
		return report, fmt.Errorf("Error retrieving network prune report: %v", err)
	}

	return report, nil
}
