// +build !exclude_graphdriver_zfs,linux !exclude_graphdriver_zfs,freebsd

package register

import (
	// register the zfs driver
	_ "github.com/tiborvass/docker/daemon/graphdriver/zfs"
)
