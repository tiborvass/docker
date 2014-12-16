package sandbox

import (
	"errors"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/docker/daemon/execdriver"
	"github.com/docker/docker/state"
)

func NewController(dr execdriver.Driver) *Controller {
	return &Controller{driver: dr}
}

// FIXME:networking Just to get things to build
type Controller struct {
	driver execdriver.Driver
}

func (c *Controller) Restore(state state.State) error {
	return nil
}

func (c *Controller) List() []string {
	return []string{}
}

func (c *Controller) Get(id string) (Sandbox, error) {
	return &NativeSandbox{
		ID:     id,
		driver: c.driver,
	}, nil
}

func (c *Controller) Remove(id string) error {
	return errors.New("Not implemented")
}

func (c *Controller) New() (string, error) {
	return "", errors.New("Not implemented")
}

type NativeSandbox struct {
	ID     string
	driver execdriver.Driver
}

func (s *NativeSandbox) AddIface(iface *execdriver.NetworkSettings) error {
	log.Printf("Sandbox %v", s)
	return s.driver.AddIface(s.ID, iface)
}

type Sandbox interface {
	//Exec(cmd string, args []string, env []string) error
	AddIface(i *execdriver.NetworkSettings) error
}
