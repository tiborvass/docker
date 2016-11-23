package main

import (
	"github.com/tiborvass/docker/pkg/integration/checker"
	"github.com/go-check/check"

	"os"
	"path/filepath"
	"strings"
)

var (
	pluginProcessName = "no-remove"
	pName             = "tiborvass/no-remove"
	pTag              = "latest"
	pNameWithTag      = pName + ":" + pTag
)

func (s *DockerSuite) TestPluginBasicOps(c *check.C) {
	testRequires(c, DaemonIsLinux, Network)
	_, _, err := dockerCmdWithError("plugin", "install", "--grant-all-permissions", pNameWithTag)
	c.Assert(err, checker.IsNil)

	out, _, err := dockerCmdWithError("plugin", "ls")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, pName)
	c.Assert(out, checker.Contains, pTag)
	c.Assert(out, checker.Contains, "true")

	id, _, err := dockerCmdWithError("plugin", "inspect", "-f", "{{.Id}}", pNameWithTag)
	id = strings.TrimSpace(id)
	c.Assert(err, checker.IsNil)

	out, _, err = dockerCmdWithError("plugin", "remove", pNameWithTag)
	c.Assert(out, checker.Contains, "is enabled")

	_, _, err = dockerCmdWithError("plugin", "disable", pNameWithTag)
	c.Assert(err, checker.IsNil)

	out, _, err = dockerCmdWithError("plugin", "remove", pNameWithTag)
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, pNameWithTag)

	_, err = os.Stat(filepath.Join(dockerBasePath, "plugins", id))
	if !os.IsNotExist(err) {
		c.Fatal(err)
	}
}

func (s *DockerSuite) TestPluginForceRemove(c *check.C) {
	testRequires(c, DaemonIsLinux, Network)
	out, _, err := dockerCmdWithError("plugin", "install", "--grant-all-permissions", pNameWithTag)
	c.Assert(err, checker.IsNil)

	out, _, err = dockerCmdWithError("plugin", "remove", pNameWithTag)
	c.Assert(out, checker.Contains, "is enabled")

	out, _, err = dockerCmdWithError("plugin", "remove", "--force", pNameWithTag)
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, pNameWithTag)
}

func (s *DockerSuite) TestPluginActive(c *check.C) {
	testRequires(c, DaemonIsLinux, Network, IsAmd64)
	out, _, err := dockerCmdWithError("plugin", "install", "--grant-all-permissions", pNameWithTag)
	c.Assert(err, checker.IsNil)

	out, _, err = dockerCmdWithError("volume", "create", "-d", pNameWithTag)
	c.Assert(err, checker.IsNil)

	vID := strings.TrimSpace(out)

	out, _, err = dockerCmdWithError("plugin", "remove", pNameWithTag)
	c.Assert(out, checker.Contains, "is in use")

	_, _, err = dockerCmdWithError("volume", "rm", vID)
	c.Assert(err, checker.IsNil)

	out, _, err = dockerCmdWithError("plugin", "remove", pNameWithTag)
	c.Assert(out, checker.Contains, "is enabled")

	_, _, err = dockerCmdWithError("plugin", "disable", pNameWithTag)
	c.Assert(err, checker.IsNil)

	out, _, err = dockerCmdWithError("plugin", "remove", pNameWithTag)
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, pNameWithTag)
}

func (s *DockerSuite) TestPluginInstallDisable(c *check.C) {
	testRequires(c, DaemonIsLinux, Network)
	out, _, err := dockerCmdWithError("plugin", "install", "--grant-all-permissions", "--disable", pName)
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Contains, pName)

	out, _, err = dockerCmdWithError("plugin", "ls")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, "false")

	out, _, err = dockerCmdWithError("plugin", "enable", pName)
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Contains, pName)

	out, _, err = dockerCmdWithError("plugin", "disable", pName)
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Contains, pName)

	out, _, err = dockerCmdWithError("plugin", "remove", pName)
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Contains, pName)
}

func (s *DockerSuite) TestPluginInstallDisableVolumeLs(c *check.C) {
	testRequires(c, DaemonIsLinux, Network)
	out, _, err := dockerCmdWithError("plugin", "install", "--grant-all-permissions", "--disable", pName)
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Contains, pName)

	dockerCmd(c, "volume", "ls")
}

func (s *DockerSuite) TestPluginSet(c *check.C) {
	testRequires(c, DaemonIsLinux, Network)
	out, _ := dockerCmd(c, "plugin", "install", "--grant-all-permissions", "--disable", pName)
	c.Assert(strings.TrimSpace(out), checker.Contains, pName)

	env, _ := dockerCmd(c, "plugin", "inspect", "-f", "{{.Settings.Env}}", pName)
	c.Assert(strings.TrimSpace(env), checker.Equals, "[DEBUG=0]")

	dockerCmd(c, "plugin", "set", pName, "DEBUG=1")

	env, _ = dockerCmd(c, "plugin", "inspect", "-f", "{{.Settings.Env}}", pName)
	c.Assert(strings.TrimSpace(env), checker.Equals, "[DEBUG=1]")
}

func (s *DockerSuite) TestPluginInstallArgs(c *check.C) {
	testRequires(c, DaemonIsLinux, Network)
	out, _ := dockerCmd(c, "plugin", "install", "--grant-all-permissions", "--disable", pName, "DEBUG=1")
	c.Assert(strings.TrimSpace(out), checker.Contains, pName)

	env, _ := dockerCmd(c, "plugin", "inspect", "-f", "{{.Settings.Env}}", pName)
	c.Assert(strings.TrimSpace(env), checker.Equals, "[DEBUG=1]")
}

func (s *DockerSuite) TestPluginInstallImage(c *check.C) {
	testRequires(c, DaemonIsLinux, Network)
	out, _, err := dockerCmdWithError("plugin", "install", "redis")
	c.Assert(err, checker.NotNil)
	c.Assert(out, checker.Contains, "content is not a plugin")
}

func (s *DockerSuite) TestPluginEnableDisableNegative(c *check.C) {
	testRequires(c, DaemonIsLinux, Network)
	out, _, err := dockerCmdWithError("plugin", "install", "--grant-all-permissions", pName)
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Contains, pName)

	out, _, err = dockerCmdWithError("plugin", "enable", pName)
	c.Assert(err, checker.NotNil)
	c.Assert(strings.TrimSpace(out), checker.Contains, "already enabled")

	_, _, err = dockerCmdWithError("plugin", "disable", pName)
	c.Assert(err, checker.IsNil)

	out, _, err = dockerCmdWithError("plugin", "disable", pName)
	c.Assert(err, checker.NotNil)
	c.Assert(strings.TrimSpace(out), checker.Contains, "already disabled")

	_, _, err = dockerCmdWithError("plugin", "remove", pName)
	c.Assert(err, checker.IsNil)
}
