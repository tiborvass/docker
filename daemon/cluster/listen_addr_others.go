// +build !linux

package cluster // import "github.com/tiborvass/docker/daemon/cluster"

import "net"

func (c *Cluster) resolveSystemAddr() (net.IP, error) {
	return c.resolveSystemAddrViaSubnetCheck()
}
