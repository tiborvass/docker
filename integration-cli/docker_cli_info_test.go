package main

import (
	"fmt"

	"github.com/tiborvass/docker/pkg/integration/checker"
	"github.com/tiborvass/docker/utils"
	"github.com/go-check/check"
)

// ensure docker info succeeds
func (s *DockerSuite) TestInfoEnsureSucceeds(c *check.C) {
	out, _ := dockerCmd(c, "info")

	// always shown fields
	stringsToCheck := []string{
		"ID:",
		"Containers:",
		"Images:",
		"Execution Driver:",
		"Logging Driver:",
		"Operating System:",
		"CPUs:",
		"Total Memory:",
		"Kernel Version:",
		"Storage Driver:",
	}

	if utils.ExperimentalBuild() {
		stringsToCheck = append(stringsToCheck, "Experimental: true")
	}

	for _, linePrefix := range stringsToCheck {
		c.Assert(out, checker.Contains, linePrefix, check.Commentf("couldn't find string %v in output", linePrefix))
	}
}

// TestInfoDiscoveryBackend verifies that a daemon run with `--cluster-advertise` and
// `--cluster-store` properly show the backend's endpoint in info output.
func (s *DockerSuite) TestInfoDiscoveryBackend(c *check.C) {
	testRequires(c, SameHostDaemon)

	d := NewDaemon(c)
	discoveryBackend := "consul://consuladdr:consulport/some/path"
	err := d.Start(fmt.Sprintf("--cluster-store=%s", discoveryBackend), "--cluster-advertise=foo")
	c.Assert(err, checker.IsNil)
	defer d.Stop()

	out, err := d.Cmd("info")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, fmt.Sprintf("Cluster store: %s\n", discoveryBackend))
}
