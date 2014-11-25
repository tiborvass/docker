package sandbox

import (
	"net"

	"github.com/docker/docker/core"
)

// FIXME:networking Just to get things to build
type Controller interface {
	List() []core.DID
	Get(id core.DID) (Sandbox, error)
	Remove(id core.DID) error
	New() (core.DID, error)
}

type Sandbox interface {
	Exec(cmd string, args []string, env []string) error
	AddNetIface(i *net.Interface)
}
