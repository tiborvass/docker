package daemon

import (
	"time"

	"github.com/tiborvass/docker/engine"
)

func (daemon *Daemon) ContainerWait(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("Usage: %s", job.Name)
	}
	name := job.Args[0]
	container, err := daemon.Get(name)
	if err != nil {
		return job.Errorf("%s: %s", job.Name, err.Error())
	}
	status, _ := container.WaitStop(-1 * time.Second)
	job.Printf("%d\n", status)
	return engine.StatusOK
}
