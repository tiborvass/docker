package engine

import (
	"fmt"
	"os"
	"log"
	"runtime"
	"github.com/dotcloud/docker/utils"
)


type Handler func(*Job) string

var globalHandlers map[string]Handler

func init() {
	globalHandlers = make(map[string]Handler)
}

func Register(name string, handler Handler) error {
	globalHandlers[name] = handler
	return nil
}

// The Engine is the core of Docker.
// It acts as a store for *containers*, and allows manipulation of these
// containers by executing *jobs*.
type Engine struct {
	root		string
	handlers	map[string]Handler
	hack		Hack	// data for temporary hackery (see hack.go)
}

// New initializes a new engine managing the directory specified at `root`.
// `root` is used to store containers and any other state private to the engine.
// Changing the contents of the root without executing a job will cause unspecified
// behavior.
func New(root string) (*Engine, error) {
	// Check for unsupported architectures
	if runtime.GOARCH != "amd64" {
		return nil, fmt.Errorf("The docker runtime currently only supports amd64 (not %s). This will change in the future. Aborting.", runtime.GOARCH)
	}
	// Check for unsupported kernel versions
	// FIXME: it would be cleaner to not test for specific versions, but rather
	// test for specific functionalities.
	// Unfortunately we can't test for the feature "does not cause a kernel panic"
	// without actually causing a kernel panic, so we need this workaround until
	// the circumstances of pre-3.8 crashes are clearer.
	// For details see http://github.com/dotcloud/docker/issues/407
	if k, err := utils.GetKernelVersion(); err != nil {
		log.Printf("WARNING: %s\n", err)
	} else {
		if utils.CompareKernelVersion(k, &utils.KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0}) < 0 {
			log.Printf("WARNING: You are running linux kernel version %s, which might be unstable running docker. Please upgrade your kernel to 3.8.0.", k.String())
		}
	}
	if err := os.MkdirAll(root, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	eng := &Engine{
		root:		root,
		handlers:	globalHandlers,
	}
	return eng, nil
}

// Job creates a new job which can later be executed.
// This function mimics `Command` from the standard os/exec package.
func (eng *Engine) Job(name string, args ...string) *Job {
	job := &Job{
		Eng:		eng,
		Name:		name,
		Args:		args,
		Stdin:		os.Stdin,
		Stdout:		os.Stdout,
		Stderr:		os.Stderr,
	}
	handler, exists := eng.handlers[name]
	if exists {
		job.handler = handler
	}
	return job
}

