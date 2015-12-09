package lib

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/pkg/parsers/filters"
)

// ContainerList returns the list of containers in the docker host.
func (cli *Client) ContainerList(options types.ContainerListOptions) ([]types.Container, error) {
	var query url.Values

	if options.All {
		query.Set("all", "1")
	}

	if options.Limit != -1 {
		query.Set("limit", strconv.Itoa(options.Limit))
	}

	if options.Since != "" {
		query.Set("since", options.Since)
	}

	if options.Before != "" {
		query.Set("before", options.Before)
	}

	if options.Size {
		query.Set("size", "1")
	}

	if options.Filter.Len() > 0 {
		filterJSON, err := filters.ToParam(options.Filter)
		if err != nil {
			return nil, err
		}

		query.Set("filters", filterJSON)
	}

	resp, err := cli.GET("/containers/json", query, nil)
	if err != nil {
		return nil, err
	}
	defer ensureReaderClosed(resp)

	var containers []types.Container
	err = json.NewDecoder(resp.body).Decode(&containers)
	return containers, err
}
