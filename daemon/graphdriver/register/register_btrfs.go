// +build !exclude_graphdriver_btrfs,linux

package register // import "github.com/tiborvass/docker/daemon/graphdriver/register"

import (
	// register the btrfs graphdriver
	_ "github.com/tiborvass/docker/daemon/graphdriver/btrfs"
)
