// +build !exclude_graphdriver_overlayfs

package daemon

import (
	_ "github.com/tiborvass/docker/daemon/graphdriver/overlayfs"
)
