package main

import (
	"github.com/go-check/check"
	"github.com/moby/moby-core/integration-cli/checker"
)

func (s *DockerSuite) TestStopContainerWithRestartPolicyAlways(c *check.C) {
	dockerCmd(c, "run", "--name", "verifyRestart1", "-d", "--restart=always", "busybox", "false")
	dockerCmd(c, "run", "--name", "verifyRestart2", "-d", "--restart=always", "busybox", "false")

	c.Assert(waitRun("verifyRestart1"), checker.IsNil)
	c.Assert(waitRun("verifyRestart2"), checker.IsNil)

	dockerCmd(c, "stop", "verifyRestart1")
	dockerCmd(c, "stop", "verifyRestart2")
}
