package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/docker/docker/extensions/simplebridge"
	state "github.com/docker/docker/extensions/simplebridge/drivertest/mockstate"
)

func create(driver *simplebridge.BridgeDriver) error {
	if err := driver.AddNetwork("test"); err != nil {
		return err
	}

	return nil
}

func destroy(driver *simplebridge.BridgeDriver) error {
	if err := driver.RemoveNetwork("test"); err != nil {
		panic(err)
	}

	return nil
}

func createEndpoint(driver *simplebridge.BridgeDriver) error {
	network := driver.GetNetwork("test")
	if network == nil {
		return errors.New("network does not exist")
	}

	if _, err := driver.Link("test", "endpointtest", nil, true); err != nil {
		return err
	}

	return nil
}

func destroyEndpoint(driver *simplebridge.BridgeDriver) error {
	network := driver.GetNetwork("test")
	if network == nil {
		return errors.New("network does not exist")
	}

	if err := driver.Unlink("test", "endpointtest", nil); err != nil {
		return err
	}

	return nil
}

func throw(s interface{}) {
	switch s.(type) {
	case error:
		fmt.Fprintln(os.Stderr, s.(error).Error())
	case string:
		fmt.Fprintln(os.Stderr, s.(string))
	default:
		panic(fmt.Errorf("%v", s))
	}

	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		throw("supply create or destroy")
	}

	mystate := state.Load()
	driver := &simplebridge.BridgeDriver{}
	if err := driver.Restore(mystate); err != nil {
		throw(err)
	}

	switch os.Args[1] {
	case "create":
		if err := create(driver); err != nil {
			throw(err)
		}
	case "create_endpoint":
		if err := createEndpoint(driver); err != nil {
			throw(err)
		}
	case "destroy_endpoint":
		if err := destroyEndpoint(driver); err != nil {
			throw(err)
		}
	case "destroy":
		if err := destroy(driver); err != nil {
			throw(err)
		}
	default:
		throw("supply create or destroy")
	}

	if err := mystate.Save(); err != nil {
		throw(err.Error())
	}

	os.Exit(0)
}
