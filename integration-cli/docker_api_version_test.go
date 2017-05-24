package main

import (
	"github.com/tiborvass/docker/client"
	"github.com/tiborvass/docker/dockerversion"
	"github.com/tiborvass/docker/integration-cli/checker"
	"github.com/go-check/check"
	"golang.org/x/net/context"
)

func (s *DockerSuite) TestGetVersion(c *check.C) {
	cli, err := client.NewEnvClient()
	c.Assert(err, checker.IsNil)
	defer cli.Close()

	v, err := cli.ServerVersion(context.Background())
	c.Assert(v.Version, checker.Equals, dockerversion.Version, check.Commentf("Version mismatch"))
}
