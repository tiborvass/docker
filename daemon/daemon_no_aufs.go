// +build exclude_graphdriver_aufs,linux

package daemon

import (
	"github.com/tiborvass/docker/daemon/graphdriver"
)

func migrateIfAufs(driver graphdriver.Driver, root string) error {
	return nil
}
