// +build !exclude_graphdriver_overlay

package daemon

import (
	_ "github.com/tiborvass/docker/daemon/graphdriver/overlay"
)
