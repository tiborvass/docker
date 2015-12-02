package daemon

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/tiborvass/docker/runconfig"
)

func TestContainerDoubleDelete(t *testing.T) {
	tmp, err := ioutil.TempDir("", "docker-daemon-unix-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	daemon := &Daemon{
		repository: tmp,
		root:       tmp,
	}
	daemon.containers = &contStore{s: make(map[string]*Container)}

	container := &Container{
		CommonContainer: CommonContainer{
			ID:     "test",
			State:  NewState(),
			Config: &runconfig.Config{},
		},
	}
	daemon.containers.Add(container.ID, container)

	// Mark the container as having a delete in progress
	if err := container.setRemovalInProgress(); err != nil {
		t.Fatal(err)
	}

	// Try to remove the container when it's start is removalInProgress.
	// It should ignore the container and not return an error.
	if err := daemon.ContainerRm(container.ID, &ContainerRmConfig{ForceRemove: true}); err != nil {
		t.Fatal(err)
	}
}
