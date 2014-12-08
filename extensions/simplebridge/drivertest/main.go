package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/extensions"
	"github.com/docker/docker/extensions/simplebridge"
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
	if _, err := driver.GetNetwork("test"); err != nil {
		return err
	}

	if _, err := driver.Link("test", "ept", nil, true); err != nil {
		return err
	}

	return nil
}

func destroyEndpoint(driver *simplebridge.BridgeDriver) error {
	if _, err := driver.GetNetwork("test"); err != nil {
		return err
	}

	if err := driver.Unlink("test", "ept", nil); err != nil {
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

	state, err := extensions.GitStateFromFolder("/tmp/drivertest", "drivertest")
	if err != nil {
		throw(err)
	}

	driver := &simplebridge.BridgeDriver{}
	if err := driver.Restore(state); err != nil {
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

	os.Exit(0)
}
