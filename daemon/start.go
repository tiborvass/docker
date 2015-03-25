package daemon

import (
	"github.com/tiborvass/docker/engine"
	"github.com/tiborvass/docker/runconfig"
)

func (daemon *Daemon) ContainerStart(job *engine.Job) engine.Status {
	if len(job.Args) < 1 {
		return job.Errorf("Usage: %s container_id", job.Name)
	}
	var (
		name = job.Args[0]
	)

	container, err := daemon.Get(name)
	if err != nil {
		return job.Error(err)
	}

	if container.IsPaused() {
		return job.Errorf("Cannot start a paused container, try unpause instead.")
	}

	if container.IsRunning() {
		return job.Errorf("Container already started")
	}

	// If no environment was set, then no hostconfig was passed.
	// This is kept for backward compatibility - hostconfig should be passed when
	// creating a container, not during start.
	if len(job.Environ()) > 0 {
		hostConfig := runconfig.ContainerHostConfigFromJob(job)
		if err := daemon.setHostConfig(container, hostConfig); err != nil {
			return job.Error(err)
		}
	}
	if err := container.Start(); err != nil {
		container.LogEvent("die")
		return job.Errorf("Cannot start container %s: %s", name, err)
	}

	return engine.StatusOK
}

func (daemon *Daemon) setHostConfig(container *Container, hostConfig *runconfig.HostConfig) error {
	container.Lock()
	defer container.Unlock()
	if err := parseSecurityOpt(container, hostConfig); err != nil {
		return err
	}

	// Register any links from the host config before starting the container
	if err := daemon.RegisterLinks(container, hostConfig); err != nil {
		return err
	}

	container.hostConfig = hostConfig
	container.toDisk()

	return nil
}
