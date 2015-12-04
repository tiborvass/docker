package lib

import (
	"encoding/json"
	"runtime"

	"github.com/tiborvass/docker/api"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/dockerversion"
	"github.com/tiborvass/docker/utils"
)

// SystemVersion returns information of the docker client and server host.
func (cli *Client) SystemVersion() (types.VersionResponse, error) {
	client := &types.Version{
		Version:      dockerversion.Version,
		APIVersion:   api.Version,
		GoVersion:    runtime.Version(),
		GitCommit:    dockerversion.GitCommit,
		BuildTime:    dockerversion.BuildTime,
		Os:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Experimental: utils.ExperimentalBuild(),
	}

	resp, err := cli.GET("/version", nil, nil)
	if err != nil {
		return types.VersionResponse{Client: client}, err
	}
	defer ensureReaderClosed(resp)

	var server types.Version
	err = json.NewDecoder(resp.body).Decode(&server)
	if err != nil {
		return types.VersionResponse{Client: client}, err
	}
	return types.VersionResponse{Client: client, Server: &server}, nil
}
