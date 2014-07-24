// +build !exclude_graphdriver_btrfs

package daemon

import (
	_ "github.com/tiborvass/docker/daemon/graphdriver/btrfs"
)
