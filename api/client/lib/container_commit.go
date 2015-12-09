package lib

import (
	"encoding/json"
	"net/url"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/runconfig"
)

// ContainerCommit applies changes into a container and creates a new tagged image.
func (cli *Client) ContainerCommit(options types.ContainerCommitOptions) (types.ContainerCommitResponse, error) {
	query := url.Values{}
	query.Set("container", options.ContainerID)
	query.Set("repo", options.RepositoryName)
	query.Set("tag", options.Tag)
	query.Set("comment", options.Comment)
	query.Set("author", options.Author)
	for _, change := range options.Changes {
		query.Add("changes", change)
	}
	if options.Pause != true {
		query.Set("pause", "0")
	}

	var (
		config   *runconfig.Config
		response types.ContainerCommitResponse
	)

	if options.JSONConfig != "" {
		config = &runconfig.Config{}
		if err := json.Unmarshal([]byte(options.JSONConfig), config); err != nil {
			return response, err
		}
	}

	resp, err := cli.post("/commit", query, config, nil)
	if err != nil {
		return response, err
	}
	defer ensureReaderClosed(resp)

	if err := json.NewDecoder(resp.body).Decode(&response); err != nil {
		return response, err
	}

	return response, nil
}
