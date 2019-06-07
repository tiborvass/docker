package container // import "github.com/tiborvass/docker/integration/container"

import (
	"context"
	"fmt"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/client"
	"github.com/tiborvass/docker/integration/internal/container"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/skip"
)

func TestCopyFromContainerPathDoesNotExist(t *testing.T) {
	defer setupTest(t)()
	skip.If(t, testEnv.OSType == "windows")

	ctx := context.Background()
	apiclient := testEnv.APIClient()
	cid := container.Create(ctx, t, apiclient)

	_, _, err := apiclient.CopyFromContainer(ctx, cid, "/dne")
	assert.Check(t, client.IsErrNotFound(err))
	expected := fmt.Sprintf("No such container:path: %s:%s", cid, "/dne")
	assert.Check(t, is.ErrorContains(err, expected))
}

func TestCopyFromContainerPathIsNotDir(t *testing.T) {
	defer setupTest(t)()
	skip.If(t, testEnv.OSType == "windows")

	ctx := context.Background()
	apiclient := testEnv.APIClient()
	cid := container.Create(ctx, t, apiclient)

	_, _, err := apiclient.CopyFromContainer(ctx, cid, "/etc/passwd/")
	assert.Assert(t, is.ErrorContains(err, "not a directory"))
}

func TestCopyToContainerPathDoesNotExist(t *testing.T) {
	defer setupTest(t)()
	skip.If(t, testEnv.OSType == "windows")

	ctx := context.Background()
	apiclient := testEnv.APIClient()
	cid := container.Create(ctx, t, apiclient)

	err := apiclient.CopyToContainer(ctx, cid, "/dne", nil, types.CopyToContainerOptions{})
	assert.Check(t, client.IsErrNotFound(err))
	expected := fmt.Sprintf("No such container:path: %s:%s", cid, "/dne")
	assert.Check(t, is.ErrorContains(err, expected))
}

func TestCopyToContainerPathIsNotDir(t *testing.T) {
	defer setupTest(t)()
	skip.If(t, testEnv.OSType == "windows")

	ctx := context.Background()
	apiclient := testEnv.APIClient()
	cid := container.Create(ctx, t, apiclient)

	err := apiclient.CopyToContainer(ctx, cid, "/etc/passwd/", nil, types.CopyToContainerOptions{})
	assert.Assert(t, is.ErrorContains(err, "not a directory"))
}
