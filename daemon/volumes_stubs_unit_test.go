// +build !experimental

package daemon

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/tiborvass/docker/volume"
	"github.com/tiborvass/docker/volume/drivers"
	"github.com/tiborvass/docker/volume/local"
)

func TestGetVolumeDefaultDriver(t *testing.T) {
	tmp, err := ioutil.TempDir("", "volume-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	l, err := local.New(tmp)
	if err != nil {
		t.Fatal(err)
	}
	volumedrivers.Register(l, volume.DefaultDriverName)
	d, err := getVolumeDriver("missing")
	if err != nil {
		t.Fatal(err)
	}

	if d.Name() != volume.DefaultDriverName {
		t.Fatalf("Expected local driver, was %s\n", d.Name)
	}
}
