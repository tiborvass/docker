package network

import "sync"

type Controller struct {
	driver    NetworkDriver
	networks  map[DID]Network
	endpoints map[DID]Endpoint
	state     state.State
	mutex     sync.Mutex
}

func NewController(s state.State, driver network.NetworkDriver) (*Controller, error) {
	return &Controller{
		state:     s,
		driver:    driver,
		networks:  map[DID]network.Network{},
		endpoints: map[DID]network.Endpoint{},
	}, nil
}

func (c *Controller) ListNetworks() ([]DID, error) {
	dids := []DID{}
	mutex.Lock()
	for did := range c.networks {
		dids = append(dids, did)
	}
	mutex.Unlock()

	return dids, nil
}

func (c *Controller) GetNetwork(id DID) (network.Network, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return b.networks[id]
}

func (c *Controller) RemoveNetwork(id DID) error {
	mutex.Lock()
	defer mutex.Unlock()

	if err := c.driver.RemoveNetwork(did, c.state.Scope(did)); err != nil {
		return err
	}

	delete(c.networks, id)

	return nil
}

func (c *Controller) NewNetwork() (network.Network, error) {
	did := GenerateDID() // func GenerateDID() DID { return DID(uuid.New()) }
	net, err := c.driver.AddNetwork(did, c.state.Scope(did))
	if err != nil {
		return nil, err
	}

	mutex.Lock()
	c.networks[did] = net
	mutex.Unlock()

	return net, nil
}

func (c *Controller) GetEndpoint(id DID) (Endpoint, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return c.endpoints[id], nil
}
