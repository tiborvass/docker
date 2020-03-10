// +build !windows

package symlink // import "github.com/tiborvass/docker/pkg/symlink"

import (
	"path/filepath"
)

func evalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func isDriveOrRoot(p string) bool {
	return p == string(filepath.Separator)
}

var isAbs = filepath.IsAbs
