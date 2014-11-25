package simplebridge

import "net"

type BridgeManager struct {
	id         DID
	bridgeName string
	hairpin    bool

	// other internal configuration
}

func (m *BridgeManager) Id() DID {
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
