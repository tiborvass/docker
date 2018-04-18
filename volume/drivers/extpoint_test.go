package drivers // import "github.com/tiborvass/docker/volume/drivers"

import (
	"testing"

	volumetestutils "github.com/tiborvass/docker/volume/testutils"
)

func TestGetDriver(t *testing.T) {
	s := NewStore(nil)
	_, err := s.GetDriver("missing")
	if err == nil {
		t.Fatal("Expected error, was nil")
	}
	s.Register(volumetestutils.NewFakeDriver("fake"), "fake")

	d, err := s.GetDriver("fake")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "fake" {
		t.Fatalf("Expected fake driver, got %s\n", d.Name())
	}
}
