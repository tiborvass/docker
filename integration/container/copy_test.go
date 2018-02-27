package container // import "github.com/tiborvass/docker/integration/container"

import (
	"context"
	"fmt"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/client"
	"github.com/tiborvass/docker/integration/internal/container"
	"github.com/tiborvass/docker/internal/testutil"
	"github.com/gotestyourself/gotestyourself/skip"
	"github.com/stretchr/testify/require"
)

func TestCopyFromContainerPathDoesNotExist(t *testing.T) {
	defer setupTest(t)()

	ctx := context.Background()
	apiclient := testEnv.APIClient()
	cid := container.Create(t, ctx, apiclient)

	_, _, err := apiclient.CopyFromContainer(ctx, cid, "/dne")
	require.True(t, client.IsErrNotFound(err))
	expected := fmt.Sprintf("No such container:path: %s:%s", cid, "/dne")
	testutil.ErrorContains(t, err, expected)
}

func TestCopyFromContainerPathIsNotDir(t *testing.T) {
	defer setupTest(t)()
	skip.If(t, testEnv.OSType == "windows")

	ctx := context.Background()
	apiclient := testEnv.APIClient()
	cid := container.Create(t, ctx, apiclient)

	_, _, err := apiclient.CopyFromContainer(ctx, cid, "/etc/passwd/")
	require.Contains(t, err.Error(), "not a directory")
}

func TestCopyToContainerPathDoesNotExist(t *testing.T) {
	defer setupTest(t)()
	skip.If(t, testEnv.OSType == "windows")

	ctx := context.Background()
	apiclient := testEnv.APIClient()
	cid := container.Create(t, ctx, apiclient)

	err := apiclient.CopyToContainer(ctx, cid, "/dne", nil, types.CopyToContainerOptions{})
	require.True(t, client.IsErrNotFound(err))
	expected := fmt.Sprintf("No such container:path: %s:%s", cid, "/dne")
	testutil.ErrorContains(t, err, expected)
}

func TestCopyToContainerPathIsNotDir(t *testing.T) {
	defer setupTest(t)()
	skip.If(t, testEnv.OSType == "windows")

	ctx := context.Background()
	apiclient := testEnv.APIClient()
	cid := container.Create(t, ctx, apiclient)

	err := apiclient.CopyToContainer(ctx, cid, "/etc/passwd/", nil, types.CopyToContainerOptions{})
	require.Contains(t, err.Error(), "not a directory")
}
