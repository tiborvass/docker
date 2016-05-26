package container

import (
	"testing"

	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/pkg/signal"
)

func TestContainerStopSignal(t *testing.T) {
	c := &Container{
		CommonContainer: CommonContainer{
			Config: &container.Config{},
		},
	}

	def, err := signal.ParseSignal(signal.DefaultStopSignal)
	if err != nil {
		t.Fatal(err)
	}

	s := c.StopSignal()
	if s != int(def) {
		t.Fatalf("Expected %v, got %v", def, s)
	}

	c = &Container{
		CommonContainer: CommonContainer{
			Config: &container.Config{StopSignal: "SIGKILL"},
		},
	}
	s = c.StopSignal()
	if s != 9 {
		t.Fatalf("Expected 9, got %v", s)
	}
}

func TestContainerStopTimeout(t *testing.T) {
	c := &Container{
		CommonContainer: CommonContainer{
			Config: &container.Config{},
		},
	}

	s := c.StopTimeout()
	if s != defaultStopTimeout {
		t.Fatalf("Expected %v, got %v", defaultStopTimeout, s)
	}

	stopTimeout := 15
	c = &Container{
		CommonContainer: CommonContainer{
			Config: &container.Config{StopTimeout: &stopTimeout},
		},
	}
	s = c.StopSignal()
	if s != 15 {
		t.Fatalf("Expected 15, got %v", s)
	}
}
