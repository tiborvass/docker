package simplebridge

import (
	"net"

	c "github.com/docker/docker/core"
	"github.com/docker/docker/network"
	"github.com/docker/docker/state"
)

type BridgeManager struct {
	id         c.DID
	bridgeName string
	hairpin    bool

	// other internal configuration
}

func (m *BridgeManager) Id() c.DID {
	return m.id
}

func (m *BridgeManager) String() string {
	return string(m.Id())
}

func (m *BridgeManager) configureEndpoint() (*network.Endpoint, error) {
	return &network.Endpoint{iface: net.InterfaceByName("eth0")}
}

func (m *BridgeManager) createBridge(s state.State) error {
	return nil
}

func (m *BridgeManager) destroyBridge(s state.State) error {
	return nil
}

func (m *BridgeManager) createInterface(ep *network.Endpoint) error {
	return nil
}

func (m *BridgeManager) destroyInterface(ep *network.Endpoint) error {
	return nil
}
