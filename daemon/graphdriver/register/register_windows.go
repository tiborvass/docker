package register

import (
	// register the windows graph drivers
	_ "github.com/moby/moby-core/daemon/graphdriver/lcow"
	_ "github.com/moby/moby-core/daemon/graphdriver/windows"
)
