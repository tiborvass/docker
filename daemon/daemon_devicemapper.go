// +build !exclude_graphdriver_devicemapper,linux

package daemon

import (
	_ "github.com/tiborvass/docker/daemon/graphdriver/devmapper"
)
