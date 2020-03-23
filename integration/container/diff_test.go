package container // import "github.com/tiborvass/docker/integration/container"

import (
	"context"
	"testing"
	"time"

	containertypes "github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/integration/internal/container"
	"github.com/tiborvass/docker/integration/internal/request"
	"github.com/tiborvass/docker/pkg/archive"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/poll"
)

func TestDiff(t *testing.T) {
	defer setupTest(t)()
	client := request.NewAPIClient(t)
	ctx := context.Background()

	cID := container.Run(t, ctx, client, container.WithCmd("sh", "-c", `mkdir /foo; echo xyzzy > /foo/bar`))

	// Wait for it to exit as cannot diff a running container on Windows, and
	// it will take a few seconds to exit. Also there's no way in Windows to
	// differentiate between an Add or a Modify, and all files are under
	// a "Files/" prefix.
	expected := []containertypes.ContainerChangeResponseItem{
		{Kind: archive.ChangeAdd, Path: "/foo"},
		{Kind: archive.ChangeAdd, Path: "/foo/bar"},
	}
	if testEnv.OSType == "windows" {
		poll.WaitOn(t, container.IsInState(ctx, client, cID, "exited"), poll.WithDelay(100*time.Millisecond), poll.WithTimeout(60*time.Second))
		expected = []containertypes.ContainerChangeResponseItem{
			{Kind: archive.ChangeModify, Path: "Files/foo"},
			{Kind: archive.ChangeModify, Path: "Files/foo/bar"},
		}
	}

	items, err := client.ContainerDiff(ctx, cID)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, items))
}
