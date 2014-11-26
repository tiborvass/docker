package network

import (
	"github.com/docker/docker/sandbox"
	"github.com/docker/docker/state"
)

type Driver interface {
	AddNetwork(netid string, netstate state.State) error
	RemoveNetwork(netid string, netstate state.State) error

	Link(netid, name string, sb sandbox.Sandbox, replace bool, netstate state.State) (Endpoint, error)
	Unlink(netid, name string, sb sandbox.Sandbox) error
}
