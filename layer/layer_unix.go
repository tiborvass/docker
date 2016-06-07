// +build linux freebsd darwin openbsd solaris

package layer

import "github.com/tiborvass/docker/pkg/stringid"

func (ls *layerStore) mountID(name string) string {
	return stringid.GenerateRandomID()
}
