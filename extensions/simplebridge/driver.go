package simplebridge

import (
	"fmt"
	"path"
	"strconv"
	"sync"

	"github.com/docker/docker/core"
	"github.com/docker/docker/network"
	"github.com/docker/docker/sandbox"
	"github.com/docker/docker/state"

	"github.com/vishvananda/netlink"
)

type BridgeDriver struct {
	endpoints map[string]*BridgeEndpoint
	networks  map[core.DID]*BridgeNetwork
	state     state.State
	mutex     sync.Mutex
}

func (d *BridgeDriver) endpointNames() []string {
	retval := []string{}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	for key := range d.endpoints {
		retval = append(retval, key)
	}

	return retval
}

func (d *BridgeDriver) Restore(s state.State) error {
	d.state = s
	d.endpoints = map[string]*BridgeEndpoint{}
	d.networks = map[core.DID]*BridgeNetwork{}

	return d.loadFromState()
}

func (d *BridgeDriver) loadFromState() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if err := d.loadNetworksFromState(); err != nil {
		return err
	}

	return d.loadEndpointsFromState()
}

func (d *BridgeDriver) loadEndpointsFromState() error {
	// XXX: locking happens in loadFromState
	endpoints, err := d.state.List("endpoints")
	if err != nil {
		return err
	}

	for _, endpoint := range endpoints {
		iface, err := d.state.Get(path.Join("endpoints", endpoint, "interfaceName"))
		if err != nil {
			return err
		}

		networkId, err := d.state.Get(path.Join("endpoints", endpoint, "networkId"))
		if err != nil {
			return err
		}

		hwAddr, err := d.state.Get(path.Join("endpoints", endpoint, "hwAddr"))
		if err != nil {
			return err
		}

		mtu, err := d.state.Get(path.Join("endpoints", endpoint, "mtu"))
		if err != nil {
			return err
		}

		mtuInt, err := strconv.ParseUint(mtu, 10, 32)
		if err != nil {
			return err
		}

		d.endpoints[endpoint] = &BridgeEndpoint{
			interfaceName: iface,
			hwAddr:        hwAddr,
			mtu:           uint(mtuInt),
			network:       d.networks[core.DID(networkId)],
		}
	}

	return nil
}

func (d *BridgeDriver) loadNetworksFromState() error {
	// XXX: locking happens in loadFromState
	networks, err := d.state.List("networks")
	if err != nil {
		return err
	}

	for _, network := range networks {
		bridge, err := d.state.Get(path.Join("networks", network, "bridgeInterface"))
		if err != nil {
			return err
		}

		bridgeLink := &netlink.Bridge{netlink.LinkAttrs{Name: bridge}}

		d.networks[core.DID(network)] = &BridgeNetwork{
			bridge: bridgeLink,
			ID:     core.DID(network),
			driver: d,
		}
	}

	return nil
}

// discovery driver? should it be hooked here or in the core?
func (d *BridgeDriver) Link(s sandbox.Sandbox, id core.DID, name string, replace bool) (network.Endpoint, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if _, ok := d.networks[id]; !ok {
		return nil, fmt.Errorf("No network for id %q", id)
	}

	ep := &BridgeEndpoint{network: d.networks[id]}

	if _, ok := d.endpoints[name]; ok && !replace {
		return nil, fmt.Errorf("Endpoint %q already taken", name)
	}

	d.endpoints[name] = ep

	if err := ep.configure(s); err != nil {
		return nil, err
	}

	return ep, nil
}

func (d *BridgeDriver) Unlink(name string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	ep, ok := d.endpoints[name]
	if !ok {
		return fmt.Errorf("No endpoint for name %q", name)
	}

	if err := ep.deconfigure(); err != nil {
		return err
	}

	delete(d.endpoints, name)

	return nil
}

func (d *BridgeDriver) AddNetwork(id core.DID) (network.Network, error) {
	bridge, err := d.createBridge(string(id))
	if err != nil {
		return nil, err
	}

	d.mutex.Lock()
	d.networks[id] = bridge
	d.mutex.Unlock()
	return bridge, nil
}

func (d *BridgeDriver) RemoveNetwork(id core.DID) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	bridge, ok := d.networks[id]
	if !ok {
		return fmt.Errorf("Network %q doesn't exist", id)
	}

	return bridge.destroy()
}

func (d *BridgeDriver) createInterface(ep *BridgeEndpoint) error  { return nil }
func (d *BridgeDriver) destroyInterface(ep *BridgeEndpoint) error { return nil }
func (d *BridgeDriver) createBridge(id string) (*BridgeNetwork, error) {
	dockerbridge := &netlink.Bridge{netlink.LinkAttrs{Name: id}}

	if err := netlink.LinkAdd(dockerbridge); err != nil {
		return nil, err
	}

	if _, err := d.state.Set(path.Join("networks", id, "bridgeInterface"), id); err != nil {
		return nil, err
	}

	return &BridgeNetwork{
		bridge: dockerbridge,
		driver: d,
	}, nil
}
