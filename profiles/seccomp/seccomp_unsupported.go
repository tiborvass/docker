// +build linux,!seccomp

package seccomp

import (
	"github.com/tiborvass/docker/api/types"
)

// DefaultProfile returns a nil pointer on unsupported systems.
func DefaultProfile() *types.Seccomp {
	return nil
}
