// +build !exclude_graphdriver_devicemapper,linux

package daemon

import (
	// register the devmapper graphdriver
	_ "github.com/tiborvass/docker/daemon/graphdriver/devmapper"
)
