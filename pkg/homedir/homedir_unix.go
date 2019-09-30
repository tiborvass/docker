// +build !windows,cgo !windows,osusergo

package homedir // import "github.com/tiborvass/docker/pkg/homedir"

import (
	"os"
	"os/user"
)

// Key returns the env var name for the user's home dir based on
// the platform being run on
func Key() string {
	return "HOME"
}

// Get returns the home directory of the current user with the help of
// environment variables depending on the target operating system.
// Returned path should be used with "path/filepath" to form new paths.
// If compiling statically, ensure the osusergo build tag is used.
// If needing to do nss lookups, do not compile statically.
func Get() string {
	home := os.Getenv(Key())
	if home == "" {
		if u, err := user.Current(); err == nil {
			return u.HomeDir
		}
	}
	return home
}

// GetShortcutString returns the string that is shortcut to user's home directory
// in the native shell of the platform running on.
func GetShortcutString() string {
	return "~"
}
