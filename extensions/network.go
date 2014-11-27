package extensions

import (
	"net"
)

// this is a network configuration provided by the extension at network creation
// time. it is not intended to be malleable by docker itself, but should be
// populated with all the generic configuration required by dockerinit to work
// with the container from a network perspective.
type Interface struct {
	net.Interface
	Addresses []string // a list of CIDR addresses to assign to the interface
	Gateway   net.IP   // the gateway the interface uses
	Routes    []Route  // routes to set on this interface
}

type Route struct {
	Target  string // The CIDR address of the route target
	Gateway string // The CIDR address of the gateway (empty string means no gateway)
}
