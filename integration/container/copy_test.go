package container // import "github.com/docker/docker/integration/container"

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/integration/internal/container"
	"github.com/docker/docker/internal/test/fakecontext"
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

func TestCopyFromContainerRoot(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType == "windows", "FIXME")
	defer setupTest(t)()

	ctx := context.Background()
	apiClient := testEnv.APIClient()

	buildCtx := fakecontext.New(t, "", fakecontext.WithDockerfile(`
		FROM busybox AS work
		RUN echo hello > /foo

		FROM scratch
		COPY --from=work /foo /
	`))

	img := "testcpfrom"
	resp, err := apiClient.ImageBuild(ctx, buildCtx.AsTarReader(t), types.ImageBuildOptions{
		Tags: []string{img},
	})

	assert.NilError(t, err)
	_, err = io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	_, _, err = apiClient.ImageInspectWithRaw(ctx, img)
	assert.NilError(t, err)

	cid := container.Create(ctx, t, apiClient, container.WithImage(img))

	rdr, _, err := apiClient.CopyFromContainer(ctx, cid, "/")
	assert.NilError(t, err)
	defer rdr.Close()

	tr := tar.NewReader(rdr)
	var found bool
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)

		if h.Name != "/foo" {
			continue
		}

		found = true
		buf := make([]byte, h.Size)
		_, err = io.ReadFull(rdr, buf)
		assert.NilError(t, err)
		assert.Equal(t, string(buf), "hello\n")

		break
	}

	assert.Assert(t, found, "foo file not found in archive")
}
