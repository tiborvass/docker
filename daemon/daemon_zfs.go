// +build !exclude_graphdriver_zfs,linux !exclude_graphdriver_zfs,freebsd

package daemon

import (
	_ "github.com/tiborvass/docker/daemon/graphdriver/zfs"
)
