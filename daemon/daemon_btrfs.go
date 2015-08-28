// +build !exclude_graphdriver_btrfs,linux

package daemon

import (
	// register the btrfs graphdriver
	_ "github.com/tiborvass/docker/daemon/graphdriver/btrfs"
)
