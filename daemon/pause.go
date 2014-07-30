package daemon

import (
	"github.com/tiborvass/docker/engine"
)

func (daemon *Daemon) ContainerPause(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("Usage: %s CONTAINER", job.Name)
	}
	name := job.Args[0]
	container := daemon.Get(name)
	if container == nil {
		return job.Errorf("No such container: %s", name)
	}
	if err := container.Pause(); err != nil {
		return job.Errorf("Cannot pause container %s: %s", name, err)
	}
	job.Eng.Job("log", "pause", container.ID, daemon.Repositories().ImageName(container.Image)).Run()
	return engine.StatusOK
}
