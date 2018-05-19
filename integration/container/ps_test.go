package container // import "github.com/tiborvass/docker/integration/container"

import (
	"context"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/filters"
	"github.com/tiborvass/docker/integration/internal/container"
	"github.com/tiborvass/docker/internal/test/request"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestPsFilter(t *testing.T) {
	defer setupTest(t)()
	client := request.NewAPIClient(t)
	ctx := context.Background()

	prev := container.Create(t, ctx, client)
	top := container.Create(t, ctx, client)
	next := container.Create(t, ctx, client)

	containerIDs := func(containers []types.Container) []string {
		var entries []string
		for _, container := range containers {
			entries = append(entries, container.ID)
		}
		return entries
	}

	f1 := filters.NewArgs()
	f1.Add("since", top)
	q1, err := client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: f1,
	})
	assert.NilError(t, err)
	assert.Check(t, is.Contains(containerIDs(q1), next))

	f2 := filters.NewArgs()
	f2.Add("before", top)
	q2, err := client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: f2,
	})
	assert.NilError(t, err)
	assert.Check(t, is.Contains(containerIDs(q2), prev))
}
