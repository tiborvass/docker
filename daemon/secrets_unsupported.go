// +build !linux,!windows

package daemon // import "github.com/tiborvass/docker/daemon"

func secretsSupported() bool {
	return false
}
