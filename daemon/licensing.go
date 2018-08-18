package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/dockerversion"
)

func (daemon *Daemon) fillLicense(v *types.Info) {
	v.ProductLicense = dockerversion.DefaultProductLicense
}
