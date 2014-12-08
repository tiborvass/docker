package simplebridge

import (
	"net"

	"github.com/docker/docker/core"
	"github.com/docker/docker/network"
	"github.com/docker/docker/sandbox"

	"github.com/vishvananda/netlink"
)

type BridgeNetwork struct {
	bridge  *netlink.Bridge
	ID      string
	driver  *BridgeDriver
	network *net.IPNet
}

func (b *BridgeNetwork) Driver() network.Driver {
	return b.driver
}

func (b *BridgeNetwork) Id() core.DID {
	return core.DID(b.ID)
}

func (b *BridgeNetwork) Name() string {
	return b.ID
}

func (b *BridgeNetwork) List() []string {
	return []string{} // FIXME finish
}

func (b *BridgeNetwork) Link(s sandbox.Sandbox, name string, replace bool) (network.Endpoint, error) {
	return b.driver.Link(b.ID, name, s, replace)
}

func (b *BridgeNetwork) Unlink(name string) error {
	return b.driver.Unlink(b.ID, name, nil)
}

func (b *BridgeNetwork) destroy() error {
	return netlink.LinkDel(b.bridge)
}
