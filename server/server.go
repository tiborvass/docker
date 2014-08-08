// DEPRECATION NOTICE. PLEASE DO NOT ADD ANYTHING TO THIS FILE.
//
// server/server.go is deprecated. We are working on breaking it up into smaller, cleaner
// pieces which will be easier to find and test. This will help make the code less
// redundant and more readable.
//
// Contributors, please don't add anything to server/server.go, unless it has the explicit
// goal of helping the deprecation effort.
//
// Maintainers, please refuse patches which add code to server/server.go.
//
// Instead try the following files:
// * For code related to local image management, try graph/
// * For code related to image downloading, uploading, remote search etc, try registry/
// * For code related to the docker daemon, try daemon/
// * For small utilities which could potentially be useful outside of Docker, try pkg/
// * For miscalleneous "util" functions which are docker-specific, try encapsulating them
//     inside one of the subsystems above. If you really think they should be more widely
//     available, are you sure you can't remove the docker dependencies and move them to
//     pkg? In last resort, you can add them to utils/ (but please try not to).

package server

import (
	"sync"
	"time"

	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/engine"
)

func (srv *Server) SetRunning(status bool) {
	srv.Lock()
	defer srv.Unlock()

	srv.running = status
}

func (srv *Server) IsRunning() bool {
	srv.RLock()
	defer srv.RUnlock()
	return srv.running
}

func (srv *Server) Close() error {
	if srv == nil {
		return nil
	}
	srv.SetRunning(false)
	done := make(chan struct{})
	go func() {
		srv.tasks.Wait()
		close(done)
	}()
	select {
	// Waiting server jobs for 15 seconds, shutdown immediately after that time
	case <-time.After(time.Second * 15):
	case <-done:
	}
	if srv.daemon == nil {
		return nil
	}
	return srv.daemon.Close()
}

type Server struct {
	sync.RWMutex
	daemon      *daemon.Daemon
	pullingPool map[string]chan struct{}
	pushingPool map[string]chan struct{}
	Eng         *engine.Engine
	running     bool
	tasks       sync.WaitGroup
}
