package daemon

import (
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/libcontainerd"
)

func (daemon *Daemon) getLibcontainerdCreateOptions(container *container.Container) (*[]libcontainerd.CreateOption, error) {
	createOptions := []libcontainerd.CreateOption{}
	createOptions = append(createOptions, &libcontainerd.FlushOption{IgnoreFlushesDuringBoot: !container.HasBeenStartedBefore})
	return &createOptions, nil
}
