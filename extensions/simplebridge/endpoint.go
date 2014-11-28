package simplebridge

import (
	"github.com/docker/docker/core"
	"github.com/docker/docker/network"
	"github.com/docker/docker/sandbox"
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

func (b *BridgeEndpoint) Expose(portspec string, publish bool) error {
	return nil
}

func (b *BridgeEndpoint) configure(s sandbox.Sandbox) error {
	return nil
}

func (b *BridgeEndpoint) deconfigure() error {
	return nil
}
