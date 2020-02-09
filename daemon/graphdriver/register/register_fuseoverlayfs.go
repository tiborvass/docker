// +build !exclude_graphdriver_fuseoverlayfs,linux

package register // import "github.com/tiborvass/docker/daemon/graphdriver/register"

import (
	// register the fuse-overlayfs graphdriver
	_ "github.com/tiborvass/docker/daemon/graphdriver/fuse-overlayfs"
)
