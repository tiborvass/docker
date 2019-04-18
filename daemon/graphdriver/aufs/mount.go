// +build linux

package aufs // import "github.com/tiborvass/docker/daemon/graphdriver/aufs"

import (
	"os/exec"

	"github.com/tiborvass/docker/pkg/mount"
)

// Unmount the target specified.
func Unmount(target string) error {
	if err := exec.Command("auplink", target, "flush").Run(); err != nil {
		logger.WithError(err).Warnf("Couldn't run auplink before unmount %s", target)
	}
	return mount.Unmount(target)
}
