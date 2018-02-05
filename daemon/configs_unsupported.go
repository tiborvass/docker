// +build !linux,!windows

package daemon // import "github.com/tiborvass/docker/daemon"

func configsSupported() bool {
	return false
}
