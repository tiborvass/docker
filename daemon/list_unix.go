// +build linux freebsd

package daemon // import "github.com/tiborvass/docker/daemon"

import "github.com/tiborvass/docker/container"

// excludeByIsolation is a platform specific helper function to support PS
// filtering by Isolation. This is a Windows-only concept, so is a no-op on Unix.
func excludeByIsolation(container *container.Snapshot, ctx *listContext) iterationAction {
	return includeContainer
}
