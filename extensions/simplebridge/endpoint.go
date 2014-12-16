package simplebridge

import (
	"fmt"
	"net"

	"github.com/docker/docker/daemon/execdriver"
	"github.com/docker/docker/network"
	"github.com/docker/docker/sandbox"

	"github.com/vishvananda/netlink"
)

type BridgeEndpoint struct {
	ID string

	bridgeVeth    *netlink.Veth
	containerVeth *netlink.Veth

	interfaceName string
	hwAddr        string
	mtu           uint

	network *BridgeNetwork
}

func (b *BridgeEndpoint) Name() string {
	return b.interfaceName
}

func (b *BridgeEndpoint) Network() network.Network {
	return b.network
}

func (b *BridgeEndpoint) Expose(portspec string, publish bool) error {
	return nil
}

func (b *BridgeEndpoint) configure(name string, s sandbox.Sandbox) error {
	intVethName := fmt.Sprintf("%s-int", name)

	// if either interface exists, bail.
	if _, err := netlink.LinkByName(name); err == nil {
		return fmt.Errorf("Link %q already exists", name)
	}

	if _, err := netlink.LinkByName(intVethName); err == nil {
		return fmt.Errorf("Link %q already exists", intVethName)
	}

	// in the strange case the bridge no longer exists, bail.
	if _, err := netlink.LinkByName(b.network.Name()); err != nil {
		return fmt.Errorf("Link %q does not exist", b.network.Name())
	}

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
		PeerName: intVethName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		fmt.Printf("netlink.LinkAdd(%v)\n", veth.LinkAttrs)
		return err
	}

	if err := netlink.LinkSetMaster(veth, b.network.bridge); err != nil {
		fmt.Printf("netlink.LinkSetMaster()\n")
		return err
	}

	ip, err := b.network.ipallocator.Allocate()
	if err != nil {
		fmt.Printf("ipallocator")
		return err
	}

	ipnet := &net.IPNet{
		IP:   ip,
		Mask: b.network.network.Mask,
	}
	mtu := b.network.bridge.MTU
	if mtu == 0 {
		mtu = int(b.mtu)
		if mtu == 0 {
			mtu = 1500
		}
	}
	ns := &execdriver.NetworkSettings{
		Name:    intVethName,
		Bridge:  b.network.bridge.Name,
		Address: ipnet.String(),
		Gateway: b.network.network.IP.String(),
		Mtu:     mtu,
	}

	return s.AddIface(ns)
}

func (b *BridgeEndpoint) deconfigure(name string) error {
	return netlink.LinkDel(&netlink.Veth{LinkAttrs: netlink.LinkAttrs{Name: name}})
}
