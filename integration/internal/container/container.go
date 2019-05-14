package container

import (
	"context"
	"runtime"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/api/types/network"
	"github.com/tiborvass/docker/client"
	"gotest.tools/assert"
)

// TestContainerConfig holds container configuration struct that
// are used in api calls.
type TestContainerConfig struct {
	Name             string
	Config           *container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

// create creates a container with the specified options
// nolint: golint
func create(t *testing.T, ctx context.Context, client client.APIClient, ops ...func(*TestContainerConfig)) (container.ContainerCreateCreatedBody, error) { // nolint: golint
	t.Helper()
	cmd := []string{"top"}
	if runtime.GOOS == "windows" {
		cmd = []string{"sleep", "240"}
	}
	config := &TestContainerConfig{
		Config: &container.Config{
			Image: "busybox",
			Cmd:   cmd,
		},
		HostConfig:       &container.HostConfig{},
		NetworkingConfig: &network.NetworkingConfig{},
	}

	for _, op := range ops {
		op(config)
	}

	return client.ContainerCreate(ctx, config.Config, config.HostConfig, config.NetworkingConfig, config.Name)
}

// Create creates a container with the specified options, asserting that there was no error
func Create(t *testing.T, ctx context.Context, client client.APIClient, ops ...func(*TestContainerConfig)) string { // nolint: golint
	c, err := create(t, ctx, client, ops...)
	assert.NilError(t, err)

	return c.ID
}

// CreateExpectingErr creates a container, expecting an error with the specified message
func CreateExpectingErr(t *testing.T, ctx context.Context, client client.APIClient, errMsg string, ops ...func(*TestContainerConfig)) { // nolint: golint
	_, err := create(t, ctx, client, ops...)
	assert.ErrorContains(t, err, errMsg)
}

// Run creates and start a container with the specified options
// nolint: golint
func Run(t *testing.T, ctx context.Context, client client.APIClient, ops ...func(*TestContainerConfig)) string { // nolint: golint
	t.Helper()
	id := Create(t, ctx, client, ops...)

	err := client.ContainerStart(ctx, id, types.ContainerStartOptions{})
	assert.NilError(t, err)

	return id
}
