package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/extensions"
	"github.com/docker/docker/extensions/simplebridge"
)

func create(driver *simplebridge.BridgeDriver, name string) error {
	if err := driver.AddNetwork(name); err != nil {
		return err
	}

	return nil
}

func destroy(driver *simplebridge.BridgeDriver, name string) error {
	if err := driver.RemoveNetwork(name); err != nil {
		panic(err)
	}

	return nil
}

func createEndpoint(driver *simplebridge.BridgeDriver, name string) error {
	if _, err := driver.GetNetwork(name); err != nil {
		return err
	}

	if _, err := driver.Link(name, "ept", nil, true); err != nil {
		return err
	}

	return nil
}

func destroyEndpoint(driver *simplebridge.BridgeDriver, name string) error {
	if _, err := driver.GetNetwork(name); err != nil {
		return err
	}

	if err := driver.Unlink(name, "ept", nil); err != nil {
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

	state, err := extensions.GitStateFromFolder("/tmp/drivertest2", "drivertest2")
	if err != nil {
		throw(err)
	}

	driver := &simplebridge.BridgeDriver{}
	if err := driver.Restore(state); err != nil {
		throw(err)
	}

	name := os.Args[2]

	switch os.Args[1] {
	case "create":
		if err := create(driver, name); err != nil {
			throw(err)
		}
	case "create_endpoint":
		if err := createEndpoint(driver, name); err != nil {
			throw(err)
		}
	case "destroy_endpoint":
		if err := destroyEndpoint(driver, name); err != nil {
			throw(err)
		}
	case "destroy":
		if err := destroy(driver, name); err != nil {
			throw(err)
		}
	default:
		throw("supply create or destroy")
	}

	os.Exit(0)
}
