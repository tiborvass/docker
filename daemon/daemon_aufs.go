// +build !exclude_graphdriver_aufs

package daemon

import (
	"github.com/tiborvass/docker/daemon/graphdriver"
	"github.com/tiborvass/docker/daemon/graphdriver/aufs"
	"github.com/tiborvass/docker/graph"
	"github.com/tiborvass/docker/pkg/log"
)

// Given the graphdriver ad, if it is aufs, then migrate it.
// If aufs driver is not built, this func is a noop.
func migrateIfAufs(driver graphdriver.Driver, root string) error {
	if ad, ok := driver.(*aufs.Driver); ok {
		log.Debugf("Migrating existing containers")
		if err := ad.Migrate(root, graph.SetupInitLayer); err != nil {
			return err
		}
	}
	return nil
}
