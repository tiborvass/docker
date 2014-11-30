package simplebridge

import (
	"fmt"
	"path"

	"github.com/docker/docker/core"
	"github.com/docker/docker/network"
	"github.com/docker/docker/sandbox"

	"github.com/vishvananda/netlink"
)

type BridgeNetwork struct {
	bridge *netlink.Bridge
	ID     core.DID
	driver *BridgeDriver
}

func (b *BridgeNetwork) Id() core.DID {
	return b.ID
}

func (b *BridgeNetwork) List() []string {
	return b.driver.endpointNames()
}

func (b *BridgeNetwork) Link(s sandbox.Sandbox, name string, replace bool) (network.Endpoint, error) {
	return b.driver.Link(s, b.ID, name, replace)
}

func (b *BridgeNetwork) Unlink(name string) error {
	return b.driver.Unlink(name)
}

func (b *BridgeNetwork) destroy() error {
	fmt.Println(b.ID)
	if _, err := b.driver.state.Remove(path.Join("networks", string(b.ID))); err != nil {
		return err
	}

	return netlink.LinkDel(b.bridge)
}
