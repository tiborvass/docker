package container

import (
	"context"
	"testing"
	"time"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/api/types/network"
	"github.com/tiborvass/docker/api/types/strslice"
	"github.com/tiborvass/docker/client"
	"github.com/tiborvass/docker/integration/internal/request"
	"github.com/gotestyourself/gotestyourself/poll"
	"github.com/gotestyourself/gotestyourself/skip"
	"github.com/stretchr/testify/require"
)

func TestKillContainerInvalidSignal(t *testing.T) {
	defer setupTest(t)()
	client := request.NewAPIClient(t)
	ctx := context.Background()
	c, err := client.ContainerCreate(ctx,
		&container.Config{
			Image: "busybox",
			Cmd:   strslice.StrSlice([]string{"top"}),
		},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		"")
	require.NoError(t, err)
	err = client.ContainerStart(ctx, c.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	err = client.ContainerKill(ctx, c.ID, "0")
	require.EqualError(t, err, "Error response from daemon: Invalid signal: 0")
	poll.WaitOn(t, containerIsInState(ctx, client, c.ID, "running"), poll.WithDelay(100*time.Millisecond))

	err = client.ContainerKill(ctx, c.ID, "SIG42")
	require.EqualError(t, err, "Error response from daemon: Invalid signal: SIG42")
	poll.WaitOn(t, containerIsInState(ctx, client, c.ID, "running"), poll.WithDelay(100*time.Millisecond))
}

func TestKillContainer(t *testing.T) {
	defer setupTest(t)()
	client := request.NewAPIClient(t)

	testCases := []struct {
		doc    string
		signal string
		status string
	}{
		{
			doc:    "no signal",
			signal: "",
			status: "exited",
		},
		{
			doc:    "non killing signal",
			signal: "SIGWINCH",
			status: "running",
		},
		{
			doc:    "killing signal",
			signal: "SIGTERM",
			status: "exited",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			ctx := context.Background()
			c, err := client.ContainerCreate(ctx,
				&container.Config{
					Image: "busybox",
					Cmd:   strslice.StrSlice([]string{"top"}),
				},
				&container.HostConfig{},
				&network.NetworkingConfig{},
				"")
			require.NoError(t, err)
			err = client.ContainerStart(ctx, c.ID, types.ContainerStartOptions{})
			require.NoError(t, err)
			err = client.ContainerKill(ctx, c.ID, tc.signal)
			require.NoError(t, err)

			poll.WaitOn(t, containerIsInState(ctx, client, c.ID, tc.status), poll.WithDelay(100*time.Millisecond))
		})
	}
}

func TestKillWithStopSignalAndRestartPolicies(t *testing.T) {
	skip.If(t, testEnv.OSType != "linux", "Windows only supports 1.25 or later")
	defer setupTest(t)()
	client := request.NewAPIClient(t)

	testCases := []struct {
		doc        string
		stopsignal string
		status     string
	}{
		{
			doc:        "same-signal-disables-restart-policy",
			stopsignal: "TERM",
			status:     "exited",
		},
		{
			doc:        "different-signal-keep-restart-policy",
			stopsignal: "CONT",
			status:     "running",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			ctx := context.Background()
			c, err := client.ContainerCreate(ctx,
				&container.Config{
					Image:      "busybox",
					Cmd:        strslice.StrSlice([]string{"top"}),
					StopSignal: tc.stopsignal,
				},
				&container.HostConfig{
					RestartPolicy: container.RestartPolicy{
						Name: "always",
					}},
				&network.NetworkingConfig{},
				"")
			require.NoError(t, err)
			err = client.ContainerStart(ctx, c.ID, types.ContainerStartOptions{})
			require.NoError(t, err)
			err = client.ContainerKill(ctx, c.ID, "TERM")
			require.NoError(t, err)

			poll.WaitOn(t, containerIsInState(ctx, client, c.ID, tc.status), poll.WithDelay(100*time.Millisecond))
		})
	}
}

func TestKillStoppedContainer(t *testing.T) {
	skip.If(t, testEnv.OSType != "linux") // Windows only supports 1.25 or later
	defer setupTest(t)()
	ctx := context.Background()
	client := request.NewAPIClient(t)
	c, err := client.ContainerCreate(ctx,
		&container.Config{
			Image: "busybox",
			Cmd:   strslice.StrSlice([]string{"top"}),
		},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		"")
	require.NoError(t, err)
	err = client.ContainerKill(ctx, c.ID, "SIGKILL")
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not running")
}

func TestKillStoppedContainerAPIPre120(t *testing.T) {
	skip.If(t, testEnv.OSType != "linux") // Windows only supports 1.25 or later
	defer setupTest(t)()
	ctx := context.Background()
	client := request.NewAPIClient(t, client.WithVersion("1.19"))
	c, err := client.ContainerCreate(ctx,
		&container.Config{
			Image: "busybox",
			Cmd:   strslice.StrSlice([]string{"top"}),
		},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		"")
	require.NoError(t, err)
	err = client.ContainerKill(ctx, c.ID, "SIGKILL")
	require.NoError(t, err)
}

func TestKillDifferentUserContainer(t *testing.T) {
	// TODO Windows: Windows does not yet support -u (Feb 2016).
	skip.If(t, testEnv.OSType != "linux", "User containers (container.Config.User) are not yet supported on %q platform", testEnv.OSType)

	defer setupTest(t)()
	ctx := context.Background()
	client := request.NewAPIClient(t, client.WithVersion("1.19"))

	cID := runSimpleContainer(ctx, t, client, "", func(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) {
		config.User = "daemon"
	})
	poll.WaitOn(t, containerIsInState(ctx, client, cID, "running"), poll.WithDelay(100*time.Millisecond))

	err := client.ContainerKill(ctx, cID, "SIGKILL")
	require.NoError(t, err)
	poll.WaitOn(t, containerIsInState(ctx, client, cID, "exited"), poll.WithDelay(100*time.Millisecond))
}
