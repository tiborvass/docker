package simplebridge

import (
	"io/ioutil"
	"testing"

	"github.com/docker/docker/state"

	"github.com/vishvananda/netlink"
)

func createNetwork(t *testing.T) *BridgeDriver {
	if link, err := netlink.LinkByName("test"); err == nil {
		netlink.LinkDel(link)
	}

	driver := &BridgeDriver{}

	dir, err := ioutil.TempDir("", "simplebridge")
	if err != nil {
		t.Fatal(err)
	}

	extensionState, err := state.GitStateFromFolder(dir, "drivertest")
	if err != nil {
		t.Fatal(err)
	}

	if err := driver.Restore(extensionState); err != nil {
		t.Fatal(err)
	}

	if err := driver.AddNetwork("test"); err != nil {
		t.Fatal(err)
	}

	return driver
}

func TestNetwork(t *testing.T) {
	driver := createNetwork(t)

	if _, err := netlink.LinkByName("test"); err != nil {
		t.Fatal(err)
	}

	if _, err := driver.GetNetwork("test"); err != nil {
		t.Fatal("Fetching network 'test' did not succeed")
	}

	if err := driver.RemoveNetwork("test"); err != nil {
		t.Fatal(err)
	}
}

func TestEndpoint(t *testing.T) {
	driver := createNetwork(t)

	if link, err := netlink.LinkByName("ept"); err == nil {
		netlink.LinkDel(link)
	}

	if _, err := driver.Link("test", "ept", nil, true); err != nil {
		t.Fatal(err)
	}

	if _, err := netlink.LinkByName("ept"); err != nil {
		t.Fatal(err)
	}

	if _, err := netlink.LinkByName("ept-int"); err != nil {
		t.Fatal(err)
	}

	if err := driver.Unlink("test", "ept", nil); err != nil {
		t.Fatal(err)
	}

	if err := driver.RemoveNetwork("test"); err != nil {
		t.Fatal(err)
	}
}
