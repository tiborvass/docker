package main

import (
	"bytes"
	"os/exec"

	"github.com/tiborvass/docker/pkg/integration/checker"
	"github.com/go-check/check"
)

func (s *DockerSuite) TestLoginWithoutTTY(c *check.C) {
	cmd := exec.Command(dockerBinary, "login")

	// Send to stdin so the process does not get the TTY
	cmd.Stdin = bytes.NewBufferString("buffer test string \n")

	// run the command and block until it's done
	err := cmd.Run()
	c.Assert(err, checker.NotNil, check.Commentf("Expected non nil err when loginning in & TTY not available"))

}
