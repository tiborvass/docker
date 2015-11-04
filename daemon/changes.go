package daemon

import "github.com/tiborvass/docker/pkg/archive"

// ContainerChanges returns a list of container fs changes
func (daemon *Daemon) ContainerChanges(name string) ([]archive.Change, error) {
	container, err := daemon.Get(name)
	if err != nil {
		return nil, err
	}

	container.Lock()
	defer container.Unlock()
	return daemon.changes(container)
}
