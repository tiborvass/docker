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
	InterfaceName string
	HWAddr        string
	MTU           uint
}

func (b *BridgeEndpoint) Name() string {
	return b.InterfaceName
}

func (b *BridgeNetwork) Id() core.DID {
	return b.ID
}

func (b *BridgeNetwork) List() []string {
	return b.driver.endpointNames()
}

func (b *BridgeNetwork) Link(s sandbox.Sandbox, name string, replace bool) (network.Endpoint, error) {
	return b.driver.Link(s, "default", name, replace) // FIXME for now
}

func (b *BridgeNetwork) Unlink(name string) error {
	return b.driver.Unlink("default")
}

func (b *BridgeNetwork) destroy(s state.State) error { return nil }
