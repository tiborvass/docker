// +build !windows,!darwin

package pidfile // import "github.com/tiborvass/docker/pkg/pidfile"

import (
	"os"
	"path/filepath"
	"strconv"
)

func processExists(pid int) bool {
	if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err == nil {
		return true
	}
	return false
}
