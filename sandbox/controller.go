package sandbox

import (
	"errors"
	"net"

	"github.com/docker/docker/state"
)

func NewController(s state.State) *Controller {
	return &Controller{}
}

// FIXME:networking Just to get things to build
type Controller struct {
}

func (c *Controller) Restore(state state.State) error {
	return nil
}

func (c *Controller) List() []string {
	return []string{}
}

func (c *Controller) Get(id string) (Sandbox, error) {
	return nil, errors.New("Not implemented")
}

func (c *Controller) Remove(id string) error {
	return errors.New("Not implemented")
}

func (c *Controller) New() (string, error) {
	return "", errors.New("Not implemented")
}

type Sandbox interface {
	Exec(cmd string, args []string, env []string) error
	AddNetIface(i *net.Interface)
}
