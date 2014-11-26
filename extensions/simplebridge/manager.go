package simplebridge

import c "github.com/docker/docker/core"

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
