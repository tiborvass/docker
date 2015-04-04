package builtins

import (
	"runtime"

	"github.com/tiborvass/docker/api"
	apiserver "github.com/tiborvass/docker/api/server"
	"github.com/tiborvass/docker/autogen/dockerversion"
	"github.com/tiborvass/docker/engine"
	"github.com/tiborvass/docker/pkg/parsers/kernel"
)

func Register(eng *engine.Engine) error {
	if err := remote(eng); err != nil {
		return err
	}
	if err := eng.Register("version", dockerVersion); err != nil {
		return err
	}

	return nil
}

// remote: a RESTful api for cross-docker communication
func remote(eng *engine.Engine) error {
	if err := eng.Register("serveapi", apiserver.ServeApi); err != nil {
		return err
	}
	return eng.Register("acceptconnections", apiserver.AcceptConnections)
}

// builtins jobs independent of any subsystem
func dockerVersion(job *engine.Job) error {
	v := &engine.Env{}
	v.SetJson("Version", dockerversion.VERSION)
	v.SetJson("ApiVersion", api.APIVERSION)
	v.SetJson("GitCommit", dockerversion.GITCOMMIT)
	v.Set("GoVersion", runtime.Version())
	v.Set("Os", runtime.GOOS)
	v.Set("Arch", runtime.GOARCH)
	if kernelVersion, err := kernel.GetKernelVersion(); err == nil {
		v.Set("KernelVersion", kernelVersion.String())
	}
	if _, err := v.WriteTo(job.Stdout); err != nil {
		return err
	}
	return nil
}
