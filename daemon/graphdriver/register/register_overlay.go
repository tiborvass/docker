// +build !exclude_graphdriver_overlay,linux

package register // import "github.com/tiborvass/docker/daemon/graphdriver/register"

import (
	// register the overlay graphdriver
	_ "github.com/tiborvass/docker/daemon/graphdriver/overlay"
)
