// +build !exclude_graphdriver_aufs,linux

package register // import "github.com/tiborvass/docker/daemon/graphdriver/register"

import (
	// register the aufs graphdriver
	_ "github.com/tiborvass/docker/daemon/graphdriver/aufs"
)
