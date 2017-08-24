package main

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/client"
	"github.com/tiborvass/docker/integration-cli/checker"
	"github.com/go-check/check"
	"golang.org/x/net/context"
)

// Test case for #22244
func (s *DockerSuite) TestAuthAPI(c *check.C) {
	testRequires(c, Network)
	config := types.AuthConfig{
		Username: "no-user",
		Password: "no-password",
	}
	cli, err := client.NewEnvClient()
	c.Assert(err, checker.IsNil)
	defer cli.Close()

	_, err = cli.RegistryLogin(context.Background(), config)
	expected := "Get https://registry-1.docker.io/v2/: unauthorized: incorrect username or password"
	c.Assert(err.Error(), checker.Contains, expected)
}
