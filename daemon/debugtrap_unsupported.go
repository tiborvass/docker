// +build !linux,!darwin,!freebsd,!windows

package daemon // import "github.com/tiborvass/docker/daemon"

func (daemon *Daemon) setupDumpStackTrap(_ string) {
	return
}
