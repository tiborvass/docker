// +build !windows

package dockerfile

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tiborvass/docker/pkg/system"
)

// normalizeDest normalizes the destination of a COPY/ADD command in a
// platform semantically consistent way.
func normalizeDest(workingDir, requested string) (string, error) {
	dest := filepath.FromSlash(requested)
	endsInSlash := strings.HasSuffix(requested, string(os.PathSeparator))
	if !system.IsAbs(requested) {
		dest = filepath.Join(string(os.PathSeparator), filepath.FromSlash(workingDir), dest)
		// Make sure we preserve any trailing slash
		if endsInSlash {
			dest += string(os.PathSeparator)
		}
	}
	return dest, nil
}

func containsWildcards(name string) bool {
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '\\' {
			i++
		} else if ch == '*' || ch == '?' || ch == '[' {
			return true
		}
	}
	return false
}

func validateCopySourcePath(imageSource *imageMount, origPath string) error {
	return nil
}
