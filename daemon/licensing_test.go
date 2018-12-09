package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/dockerversion"
	"gotest.tools/assert"
)

func TestFillLicense(t *testing.T) {
	v := &types.Info{}
	d := &Daemon{
		root: "/var/lib/docker/",
	}
	d.fillLicense(v)
	assert.Assert(t, v.ProductLicense == dockerversion.DefaultProductLicense)
}
