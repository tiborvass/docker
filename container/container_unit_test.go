package container

import (
	"testing"

	"github.com/tiborvass/docker/pkg/signal"
	"github.com/tiborvass/docker/runconfig"
)

func TestContainerStopSignal(t *testing.T) {
	c := &Container{
		CommonContainer: CommonContainer{
			Config: &runconfig.Config{},
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
			Config: &runconfig.Config{StopSignal: "SIGKILL"},
		},
	}
	s = c.StopSignal()
	if s != 9 {
		t.Fatalf("Expected 9, got %v", s)
	}
}
