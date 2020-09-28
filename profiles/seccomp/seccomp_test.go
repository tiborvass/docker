// +build linux

package seccomp // import "github.com/tiborvass/docker/profiles/seccomp"

import (
	"io/ioutil"
	"testing"

	"github.com/tiborvass/docker/oci"
)

func TestLoadProfile(t *testing.T) {
	f, err := ioutil.ReadFile("fixtures/example.json")
	if err != nil {
		t.Fatal(err)
	}
	rs := oci.DefaultSpec()
	if _, err := LoadProfile(string(f), &rs); err != nil {
		t.Fatal(err)
	}
}

// TestLoadLegacyProfile tests loading a seccomp profile in the old format
// (before https://github.com/docker/docker/pull/24510)
func TestLoadLegacyProfile(t *testing.T) {
	f, err := ioutil.ReadFile("fixtures/default-old-format.json")
	if err != nil {
		t.Fatal(err)
	}
	rs := oci.DefaultSpec()
	if _, err := LoadProfile(string(f), &rs); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDefaultProfile(t *testing.T) {
	f, err := ioutil.ReadFile("default.json")
	if err != nil {
		t.Fatal(err)
	}
	rs := oci.DefaultSpec()
	if _, err := LoadProfile(string(f), &rs); err != nil {
		t.Fatal(err)
	}
}
