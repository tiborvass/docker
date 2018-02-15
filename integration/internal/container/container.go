package container

import (
	"context"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/api/types/network"
	"github.com/tiborvass/docker/client"
	"github.com/stretchr/testify/require"
)

// TestContainerConfig holds container configuration struct that
// are used in api calls.
type TestContainerConfig struct {
	Name             string
	Config           *container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

// Create creates a container with the specified options
func Create(t *testing.T, ctx context.Context, client client.APIClient, ops ...func(*TestContainerConfig)) string { // nolint: golint
	t.Helper()
	config := &TestContainerConfig{
		Config: &container.Config{
			Image: "busybox",
			Cmd:   []string{"top"},
		},
		HostConfig:       &container.HostConfig{},
		NetworkingConfig: &network.NetworkingConfig{},
	}

	for _, op := range ops {
		op(config)
	}

	c, err := client.ContainerCreate(ctx, config.Config, config.HostConfig, config.NetworkingConfig, config.Name)
	require.NoError(t, err)

	return c.ID
}

// Run creates and start a container with the specified options
func Run(t *testing.T, ctx context.Context, client client.APIClient, ops ...func(*TestContainerConfig)) string { // nolint: golint
	t.Helper()
	id := Create(t, ctx, client, ops...)

	err := client.ContainerStart(ctx, id, types.ContainerStartOptions{})
	require.NoError(t, err)

	return id
}