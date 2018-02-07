// +build !linux

package daemon // import "github.com/tiborvass/docker/daemon"

func ensureDefaultAppArmorProfile() error {
	return nil
}
