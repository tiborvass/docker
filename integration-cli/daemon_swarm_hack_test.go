package main

import (
	"github.com/tiborvass/docker/integration-cli/daemon"
	"github.com/go-check/check"
)

func (s *DockerSwarmSuite) getDaemon(c *testing.T, nodeID string) *daemon.Daemon {
	s.daemonsLock.Lock()
	defer s.daemonsLock.Unlock()
	for _, d := range s.daemons {
		if d.NodeID() == nodeID {
			return d
		}
	}
	c.Fatalf("could not find node with id: %s", nodeID)
	return nil
}

// nodeCmd executes a command on a given node via the normal docker socket
func (s *DockerSwarmSuite) nodeCmd(c *testing.T, id string, args ...string) (string, error) {
	return s.getDaemon(c, id).Cmd(args...)
}
