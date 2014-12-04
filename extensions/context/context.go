package context

import (
	"github.com/docker/docker/network"
	"github.com/docker/docker/state"
)

type Core interface {
	RegisterNetworkDriver(driver network.Driver, name string) error
	UnregisterNetworkDriver(name string) error
}

// Additionnally to providing a scoped context, Context implements all the
// necessary methods for interacting with Docker core facilities.
type Context interface {
	Core

	MyState() state.State
	MyConfig() state.State
}

type emptyContext int

func (*emptyContext) MyState() state.State {
	return nil
}

func (*emptyContext) MyConfig() state.State {
	return nil
}

func (*emptyContext) Done() <-chan struct{} {
	return nil
}

func (*emptyContext) RegisterNetworkDriver(driver network.Driver, name string) error {
	return nil
}

func (*emptyContext) UnregisterNetworkDriver(name string) error {
	return nil
}

var rootContext = new(emptyContext)

// This is called `Background` in Google's package.
func Root() Context {
	return rootContext
}
