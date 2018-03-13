package container

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/integration/internal/container"
	"github.com/tiborvass/docker/integration/internal/request"
	"github.com/tiborvass/docker/pkg/stdcopy"
	"github.com/gotestyourself/gotestyourself/assert"
)

// Regression test for #35370
// Makes sure that when following we don't get an EOF error when there are no logs
func TestLogsFollowTailEmpty(t *testing.T) {
	defer setupTest(t)()
	client := request.NewAPIClient(t)
	ctx := context.Background()

	id := container.Run(t, ctx, client, container.WithCmd("sleep", "100000"))

	logs, err := client.ContainerLogs(ctx, id, types.ContainerLogsOptions{ShowStdout: true, Tail: "2"})
	if logs != nil {
		defer logs.Close()
	}
	assert.Check(t, err)

	_, err = stdcopy.StdCopy(ioutil.Discard, ioutil.Discard, logs)
	assert.Check(t, err)
}
