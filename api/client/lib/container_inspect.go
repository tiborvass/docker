package lib

import (
	"encoding/json"

	"github.com/tiborvass/docker/api/types"
)

// ContainerInspect returns the all the container information.
func (cli *Client) ContainerInspect(containerID string) (types.ContainerJSON, error) {
	serverResp, err := cli.GET("/containers/"+containerID+"/json", nil, nil)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	defer serverResp.body.Close()

	var response types.ContainerJSON
	json.NewDecoder(serverResp.body).Decode(&response)
	return response, err
}
