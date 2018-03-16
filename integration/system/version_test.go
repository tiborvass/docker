package system // import "github.com/tiborvass/docker/integration/system"

import (
	"testing"

	"github.com/tiborvass/docker/integration/internal/request"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"golang.org/x/net/context"
)

func TestVersion(t *testing.T) {
	client := request.NewAPIClient(t)

	version, err := client.ServerVersion(context.Background())
	assert.NilError(t, err)

	assert.Check(t, version.APIVersion != nil)
	assert.Check(t, version.Version != nil)
	assert.Check(t, version.MinAPIVersion != nil)
	assert.Check(t, is.Equal(testEnv.DaemonInfo.ExperimentalBuild, version.Experimental))
	assert.Check(t, is.Equal(testEnv.OSType, version.Os))
}
