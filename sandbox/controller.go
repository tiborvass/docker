package sandbox

import (
	"errors"
	"net"

	"github.com/docker/docker/core"
	"github.com/docker/docker/state"
)

func NewController() *Controller {
	return &Controller{}
}

// FIXME:networking Just to get things to build
type Controller struct {
}

func (c *Controller) Restore(state state.State) error {
	return nil
}

func (c *Controller) List() []core.DID {
	return []core.DID{}
}

func (c *Controller) Get(id core.DID) (Sandbox, error) {
	return nil, errors.New("Not implemented")
}

func (c *Controller) Remove(id core.DID) error {
	return errors.New("Not implemented")
}

func (c *Controller) New() (core.DID, error) {
	return "", errors.New("Not implemented")
}

type Sandbox interface {
	Exec(cmd string, args []string, env []string) error
	AddNetIface(i *net.Interface)
}
