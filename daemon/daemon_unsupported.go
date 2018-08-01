// +build !linux,!freebsd,!windows

package daemon // import "github.com/tiborvass/docker/daemon"
import "github.com/tiborvass/docker/daemon/config"

const platformSupported = false

func setupResolvConf(config *config.Config) {
}
