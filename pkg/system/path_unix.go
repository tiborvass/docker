// +build !windows

package system // import "github.com/tiborvass/docker/pkg/system"

// GetLongPathName converts Windows short pathnames to full pathnames.
// For example C:\Users\ADMIN~1 --> C:\Users\Administrator.
// It is a no-op on non-Windows platforms
func GetLongPathName(path string) (string, error) {
	return path, nil
}
