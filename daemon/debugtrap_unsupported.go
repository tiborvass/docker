// +build !linux,!darwin,!freebsd,!windows

package daemon // import "github.com/tiborvass/docker/daemon"

func (d *Daemon) setupDumpStackTrap(_ string) {
	return
}
