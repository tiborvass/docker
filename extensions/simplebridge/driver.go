package simplebridge

import "errors"

type BridgeDriver struct {
	endpoints map[DID]network.Endpoint
	network   map[DID]network.Network
}

// discovery driver? should it be hooked here or in the core?
func (d *BridgeDriver) Link(s sandbox.Sandbox, id DID, name string, replace bool) (*network.Endpoint, error) {
	ep, err := d.network[id].configureEndpoint()
	if err != nil {
		return nil, err
	}

	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := d.endpoints[name]; ok && !replace {
		return errors.New("Endpoint %q already taken", name)
	}

	d.endpoints[name] = ep

	if err := d.createInterface(ep); err != nil { // or something
		return nil, err
	}

	return ep, nil
}

func (d *BridgeDriver) Unlink(name string) error {
	return n.destroyInterface(d.endpoints[name])
}

func (d *BridgeDriver) AddNetwork(id DID, s state.State) error {
	net := &BridgeManager{id: id}
	if err := net.createBridge(s); err != nil { // use state here for parameters
		return nil, err
	}

	mutex.Lock()
	d.networks[id] = net
	mutex.Unlock()
	return nil
}

func (d *BridgeDriver) RemoveNetwork(id DID, s state.State) error {
	mutex.Lock()
	net, ok := d.networks[id]
	mutex.Unlock()

	if !ok {
		return errors.New("Network %q doesn't exist for this driver", id)
	}

	return net.destroyBridge(s)
}
