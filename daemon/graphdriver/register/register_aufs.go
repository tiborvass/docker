// +build !exclude_graphdriver_aufs,linux

package register

import (
	// register the aufs graphdriver
	_ "github.com/moby/moby-core/daemon/graphdriver/aufs"
)
