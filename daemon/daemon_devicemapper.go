// +build !exclude_graphdriver_devicemapper

package daemon

import (
	_ "github.com/tiborvass/docker/daemon/graphdriver/devmapper"
)
