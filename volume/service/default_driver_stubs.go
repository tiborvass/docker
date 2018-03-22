// +build !linux,!windows

package service // import "github.com/tiborvass/docker/volume/service"

import (
	"github.com/tiborvass/docker/pkg/idtools"
	"github.com/tiborvass/docker/volume/drivers"
)

func setupDefaultDriver(_ *drivers.Store, _ string, _ idtools.IDPair) error { return nil }
