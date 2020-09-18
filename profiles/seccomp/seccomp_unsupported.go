// +build linux,!seccomp

package seccomp // import "github.com/tiborvass/docker/profiles/seccomp"

// DefaultProfile returns a nil pointer on unsupported systems.
func DefaultProfile() *Seccomp {
	return nil
}
