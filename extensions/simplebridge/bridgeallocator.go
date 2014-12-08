package simplebridge

import (
	"errors"
	"net"

	"github.com/j-keck/arping"
)

var bridgeAddrs = []string{
	// Here we don't follow the convention of using the 1st IP of the range for the gateway.
	// This is to use the same gateway IPs as the /24 ranges, which predate the /16 ranges.
	// In theory this shouldn't matter - in practice there's bound to be a few scripts relying
	// on the internal addressing or other stupid things like that.
	// They shouldn't, but hey, let's not break them unless we really have to.
	//
	// Don't use 172.16.0.0/16, it conflicts with EC2 DNS 172.16.0.23
	//
	"10.0.42.1/16",
	"10.1.42.1/16",
	"10.42.42.1/16",

	// XXX this next line was changed from a /16 to /24 because the netmask would
	// allow for EC2's DNS to be trumped still. The 10.x/16's were moved to the top
	// as a result of this.
	"172.17.42.1/24",
	"172.16.42.1/24",
	"172.16.43.1/24",
	"172.16.44.1/24",
	"10.0.42.1/24",
	"10.0.43.1/24",
	"192.168.42.1/24",
	"192.168.43.1/24",
	"192.168.44.1/24",
}

// FIXME have this accept state objects to get at parameter data
func GetBridgeIP() (net.IP, *net.IPNet, error) {
	for _, addr := range bridgeAddrs {
		ip, ipnet, err := net.ParseCIDR(addr)
		if err != nil { // this should not happen since the list is above and static
			return nil, nil, err
		}

		// ping the IP using ARP on all interfaces to determine if we know about
		// this network already.
		if _, _, err := arping.Ping(ip); err != nil {
			switch err {
			case arping.ErrTimeout: // this is what we want
				return ip, ipnet, nil
			default:
				return nil, nil, err
			}
		}
	}

	return nil, nil, errors.New("Could not find a suitable bridge IP!")
}
