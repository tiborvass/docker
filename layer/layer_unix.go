// +build linux freebsd darwin openbsd solaris

package layer

import "github.com/moby/moby-core/pkg/stringid"

func (ls *layerStore) mountID(name string) string {
	return stringid.GenerateRandomID()
}
