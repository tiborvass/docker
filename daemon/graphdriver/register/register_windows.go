package register // import "github.com/tiborvass/docker/daemon/graphdriver/register"

import (
	// register the windows graph drivers
	_ "github.com/tiborvass/docker/daemon/graphdriver/lcow"
	_ "github.com/tiborvass/docker/daemon/graphdriver/windows"
)
