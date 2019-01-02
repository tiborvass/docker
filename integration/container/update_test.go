package container // import "github.com/tiborvass/docker/integration/container"

import (
	"context"
	"testing"
	"time"

	containertypes "github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/integration/internal/container"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/poll"
)

func TestUpdateRestartPolicy(t *testing.T) {
	defer setupTest(t)()
	client := testEnv.APIClient()
	ctx := context.Background()

	cID := container.Run(t, ctx, client, container.WithCmd("sh", "-c", "sleep 1 && false"), func(c *container.TestContainerConfig) {
		c.HostConfig.RestartPolicy = containertypes.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 3,
		}
	})

	_, err := client.ContainerUpdate(ctx, cID, containertypes.UpdateConfig{
		RestartPolicy: containertypes.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 5,
		},
	})
	assert.NilError(t, err)

	timeout := 60 * time.Second
	if testEnv.OSType == "windows" {
		timeout = 180 * time.Second
	}

	poll.WaitOn(t, container.IsInState(ctx, client, cID, "exited"), poll.WithDelay(100*time.Millisecond), poll.WithTimeout(timeout))

	inspect, err := client.ContainerInspect(ctx, cID)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(inspect.RestartCount, 5))
	assert.Check(t, is.Equal(inspect.HostConfig.RestartPolicy.MaximumRetryCount, 5))
}

func TestUpdateRestartWithAutoRemove(t *testing.T) {
	defer setupTest(t)()
	client := testEnv.APIClient()
	ctx := context.Background()

	cID := container.Run(t, ctx, client, func(c *container.TestContainerConfig) {
		c.HostConfig.AutoRemove = true
	})

	_, err := client.ContainerUpdate(ctx, cID, containertypes.UpdateConfig{
		RestartPolicy: containertypes.RestartPolicy{
			Name: "always",
		},
	})
	assert.Check(t, is.ErrorContains(err, "Restart policy cannot be updated because AutoRemove is enabled for the container"))
}
