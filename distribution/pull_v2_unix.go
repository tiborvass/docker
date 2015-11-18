// +build !windows

package distribution

import (
	"github.com/docker/distribution/manifest/schema1"
	"github.com/tiborvass/docker/image"
)

func detectBaseLayer(is image.Store, m *schema1.Manifest, rootFS *image.RootFS) error {
	return nil
}
