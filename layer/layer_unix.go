// +build linux freebsd darwin openbsd solaris

package layer

import "github.com/tiborvass/docker/pkg/stringid"

func (ls *layerStore) mountID(name string) string {
	return stringid.GenerateRandomID()
}

// ForeignSourceSupported returns whether layers downloaded from foreign sources are
// supported in this daemon.
func ForeignSourceSupported() bool {
	return false
}
