package daemon

import(
	"testing"

	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/filters"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestListInvalidFilter(t *testing.T) {
	db, err := container.NewViewDB()
	assert.Assert(t, err == nil)
	d := &Daemon{
		containersReplica: db,
	}

	f := filters.NewArgs(filters.Arg("invalid", "foo"))

	_, err = d.Containers(&types.ContainerListOptions{
		Filters: f,
	})
	assert.Assert(t, is.Error(err, "Invalid filter 'invalid'"))
}