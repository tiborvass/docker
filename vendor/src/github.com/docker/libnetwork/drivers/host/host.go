package host

import (
	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/types"
)

const networkType = "host"

type driver struct{}

// Init registers a new instance of host driver
func Init(dc driverapi.DriverCallback) error {
	return dc.RegisterDriver(networkType, &driver{})
}

func (d *driver) Config(option map[string]interface{}) error {
	return nil
}

func (d *driver) CreateNetwork(id types.UUID, option map[string]interface{}) error {
	return nil
}

func (d *driver) DeleteNetwork(nid types.UUID) error {
	return nil
}

func (d *driver) CreateEndpoint(nid, eid types.UUID, epInfo driverapi.EndpointInfo, epOptions map[string]interface{}) error {
	return nil
}

func (d *driver) DeleteEndpoint(nid, eid types.UUID) error {
	return nil
}

func (d *driver) EndpointOperInfo(nid, eid types.UUID) (map[string]interface{}, error) {
	return make(map[string]interface{}, 0), nil
}

// Join method is invoked when a Sandbox is attached to an endpoint.
func (d *driver) Join(nid, eid types.UUID, sboxKey string, jinfo driverapi.JoinInfo, options map[string]interface{}) error {
	return (jinfo.SetHostsPath("/etc/hosts"))
}

// Leave method is invoked when a Sandbox detaches from an endpoint.
func (d *driver) Leave(nid, eid types.UUID) error {
	return nil
}

func (d *driver) Type() string {
	return networkType
}
