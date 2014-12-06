package simplebridge

import (
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type IPAllocator struct {
	bridgeName string
	bridgeNet  *net.IPNet
	ipMap      map[string]struct{}
	lastIP     net.IP
	v6         bool
	mutex      sync.Mutex
}

func NewIPAllocator(bridgeName string, bridgeNet *net.IPNet) (*IPAllocator, error) {
	ip := &IPAllocator{
		bridgeName: bridgeName,
		bridgeNet:  bridgeNet,
		lastIP:     bridgeNet.IP,
		v6:         bridgeNet.IP.To4() == nil,
		ipMap:      map[string]struct{}{},
	}

	if err := ip.init(); err != nil {
		return nil, err
	}

	return ip, nil
}

func (ip *IPAllocator) Refresh() error {
	ip.mutex.Lock()
	defer ip.mutex.Unlock()

	_if, err := net.InterfaceByName(ip.bridgeName)
	if err != nil {
		return err
	}

	var list []netlink.Neigh

	if ip.v6 {
		list, err = netlink.NeighList(_if.Index, netlink.FAMILY_V6)
		if err != nil {
			return err
		}
	} else {
		list, err = netlink.NeighList(_if.Index, netlink.FAMILY_V4)
		if err != nil {
			return err
		}
	}

	for _, entry := range list {
		ip.ipMap[entry.String()] = struct{}{}
	}

	return nil
}

func (ip *IPAllocator) Allocate() (net.IP, error) {
	// FIXME use netlink package to insert into the neighbors table / arp cache
	ip.mutex.Lock()
	defer ip.mutex.Unlock()

	var (
		newip  net.IP
		ok     bool
		cycled bool
	)

	lastip := ip.bridgeNet.IP

	for {
		rawip := ipToBigInt(lastip)

		rawip.Add(rawip, big.NewInt(1))
		newip = bigIntToIP(rawip)

		if !ip.bridgeNet.Contains(newip) {
			if cycled {
				return nil, fmt.Errorf("Could not find a suitable IP for network %q", ip.bridgeNet.String())
			}

			lastip = ip.bridgeNet.IP
			cycled = true
		}

		_, ok = ip.ipMap[newip.String()]
		if !ok {
			ip.ipMap[newip.String()] = struct{}{}
			ip.lastIP = newip
			break
		}

		lastip = newip
	}

	return newip, nil
}

func (ip *IPAllocator) init() error {
	if err := ip.Refresh(); err != nil {
		return err
	}

	go func(ip *IPAllocator) {
		if err := ip.Refresh(); err != nil {
			// things that should never happen
			panic(err)
		}
		time.Sleep(10 * time.Millisecond)
	}(ip)

	return nil
}

// Converts a 4 bytes IP into a 128 bit integer
func ipToBigInt(ip net.IP) *big.Int {
	x := big.NewInt(0)
	if ip4 := ip.To4(); ip4 != nil {
		return x.SetBytes(ip4)
	}
	if ip6 := ip.To16(); ip6 != nil {
		return x.SetBytes(ip6)
	}

	log.Errorf("ipToBigInt: Wrong IP length! %s", ip)
	return nil
}

// Converts 128 bit integer into a 4 bytes IP address
func bigIntToIP(v *big.Int) net.IP {
	return net.IP(v.Bytes())
}
