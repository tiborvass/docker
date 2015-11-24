// +build !windows

package distribution

import (
	"github.com/docker/distribution/manifest/schema1"
	"github.com/tiborvass/docker/image"
)

func setupBaseLayer(history []schema1.History, rootFS image.RootFS) error {
	return nil
}
