// +build !exclude_graphdriver_aufs,linux

package daemon

import (
	// register the aufs graphdriver
	_ "github.com/tiborvass/docker/daemon/graphdriver/aufs"
)
