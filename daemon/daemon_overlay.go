// +build !exclude_graphdriver_overlay,linux

package daemon

import (
	_ "github.com/tiborvass/docker/daemon/graphdriver/overlay"
)
