package simplebridge

import (
	"github.com/docker/docker/core"
	"github.com/docker/docker/network"
	"github.com/docker/docker/sandbox"
	"github.com/docker/docker/state"
)

type BridgeNetwork struct {
	Bridge string
	ID     core.DID
	driver *BridgeDriver
}

type BridgeEndpoint struct {
	interfaceName string
	hwAddr        string
	mtu           uint
	network       *BridgeNetwork
}

func (b *BridgeEndpoint) Name() string {
	return b.interfaceName
}

func (b *BridgeEndpoint) Network() network.Network {
	return b.network
}

func (b *BridgeEndpoint) configure(s sandbox.Sandbox) error {
	return nil
}

func (b *BridgeEndpoint) deconfigure() error {
	return nil
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

func (b *BridgeNetwork) destroy(s state.State) error { return nil }
