package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/extensions/simplebridge"
	state "github.com/docker/docker/extensions/simplebridge/drivertest/mockstate"
)

func create(driver *simplebridge.BridgeDriver) error {
	if _, err := driver.AddNetwork("test", nil); err != nil {
		return err
	}

	return nil
}

func destroy(driver *simplebridge.BridgeDriver) error {
	if err := driver.RemoveNetwork("test", nil); err != nil {
		panic(err)
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
	driver := simplebridge.NewBridgeDriver(mystate)

	switch os.Args[1] {
	case "create":
		if err := create(driver); err != nil {
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
