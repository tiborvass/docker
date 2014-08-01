package daemon

import (
	"io"

	"github.com/tiborvass/docker/engine"
)

func (daemon *Daemon) ContainerExport(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("Usage: %s container_id", job.Name)
	}
	name := job.Args[0]
	if container := daemon.Get(name); container != nil {
		data, err := container.Export()
		if err != nil {
			return job.Errorf("%s: %s", name, err)
		}
		defer data.Close()

		// Stream the entire contents of the container (basically a volatile snapshot)
		if _, err := io.Copy(job.Stdout, data); err != nil {
			return job.Errorf("%s: %s", name, err)
		}
		// FIXME: factor job-specific LogEvent to engine.Job.Run()
		job.Eng.Job("log", "export", container.ID, daemon.Repositories().ImageName(container.Image)).Run()
		return engine.StatusOK
	}
	return job.Errorf("No such container: %s", name)
}
