package simplebridge

type BridgeManager struct {
	id         string
	bridgeName string
	hairpin    bool

	// other internal configuration
}

func (m *BridgeManager) Id() string {
	return m.id
}
