package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/dockerversion"
	"gotest.tools/v3/assert"
)

func TestFillLicense(t *testing.T) {
	v := &types.Info{}
	d := &Daemon{
		root: "/var/lib/docker/",
	}
	d.fillLicense(v)
	assert.Assert(t, v.ProductLicense == dockerversion.DefaultProductLicense)
}
