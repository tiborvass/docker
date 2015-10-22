package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/pkg/stringutils"
	"github.com/go-check/check"
)

func (s *DockerSuite) TestInspectApiContainerResponse(c *check.C) {
	testRequires(c, DaemonIsLinux)
	out, _ := dockerCmd(c, "run", "-d", "busybox", "true")

	cleanedContainerID := strings.TrimSpace(out)
	keysBase := []string{"Id", "State", "Created", "Path", "Args", "Config", "Image", "NetworkSettings",
		"ResolvConfPath", "HostnamePath", "HostsPath", "LogPath", "Name", "Driver", "ExecDriver", "MountLabel", "ProcessLabel", "GraphDriver"}

	cases := []struct {
		version string
		keys    []string
	}{
		{"1.20", append(keysBase, "Mounts")},
		{"1.19", append(keysBase, "Volumes", "VolumesRW")},
	}

	for _, cs := range cases {
		endpoint := fmt.Sprintf("/v%s/containers/%s/json", cs.version, cleanedContainerID)

		status, body, err := sockRequest("GET", endpoint, nil)
		c.Assert(err, check.IsNil)
		c.Assert(status, check.Equals, http.StatusOK)

		var inspectJSON map[string]interface{}
		if err = json.Unmarshal(body, &inspectJSON); err != nil {
			c.Fatalf("unable to unmarshal body for version %s: %v", cs.version, err)
		}

		for _, key := range cs.keys {
			if _, ok := inspectJSON[key]; !ok {
				c.Fatalf("%s does not exist in response for version %s", key, cs.version)
			}
		}

		//Issue #6830: type not properly converted to JSON/back
		if _, ok := inspectJSON["Path"].(bool); ok {
			c.Fatalf("Path of `true` should not be converted to boolean `true` via JSON marshalling")
		}
	}
}

func (s *DockerSuite) TestInspectApiContainerVolumeDriverLegacy(c *check.C) {
	out, _ := dockerCmd(c, "run", "-d", "busybox", "true")

	cleanedContainerID := strings.TrimSpace(out)

	cases := []string{"1.19", "1.20"}
	for _, version := range cases {
		endpoint := fmt.Sprintf("/v%s/containers/%s/json", version, cleanedContainerID)
		status, body, err := sockRequest("GET", endpoint, nil)
		c.Assert(err, check.IsNil)
		c.Assert(status, check.Equals, http.StatusOK)

		var inspectJSON map[string]interface{}
		if err = json.Unmarshal(body, &inspectJSON); err != nil {
			c.Fatalf("unable to unmarshal body for version %s: %v", version, err)
		}

		config, ok := inspectJSON["Config"]
		if !ok {
			c.Fatal("Unable to find 'Config'")
		}
		cfg := config.(map[string]interface{})
		if _, ok := cfg["VolumeDriver"]; !ok {
			c.Fatalf("Api version %s expected to include VolumeDriver in 'Config'", version)
		}
	}
}

func (s *DockerSuite) TestInspectApiContainerVolumeDriver(c *check.C) {
	out, _ := dockerCmd(c, "run", "-d", "busybox", "true")

	cleanedContainerID := strings.TrimSpace(out)

	endpoint := fmt.Sprintf("/v1.21/containers/%s/json", cleanedContainerID)
	status, body, err := sockRequest("GET", endpoint, nil)
	c.Assert(err, check.IsNil)
	c.Assert(status, check.Equals, http.StatusOK)

	var inspectJSON map[string]interface{}
	if err = json.Unmarshal(body, &inspectJSON); err != nil {
		c.Fatalf("unable to unmarshal body for version 1.21: %v", err)
	}

	config, ok := inspectJSON["Config"]
	if !ok {
		c.Fatal("Unable to find 'Config'")
	}
	cfg := config.(map[string]interface{})
	if _, ok := cfg["VolumeDriver"]; ok {
		c.Fatal("Api version 1.21 expected to not include VolumeDriver in 'Config'")
	}

	config, ok = inspectJSON["HostConfig"]
	if !ok {
		c.Fatal("Unable to find 'HostConfig'")
	}
	cfg = config.(map[string]interface{})
	if _, ok := cfg["VolumeDriver"]; !ok {
		c.Fatal("Api version 1.21 expected to include VolumeDriver in 'HostConfig'")
	}
}

func (s *DockerSuite) TestInspectApiImageResponse(c *check.C) {
	dockerCmd(c, "tag", "busybox:latest", "busybox:mytag")

	endpoint := "/images/busybox/json"
	status, body, err := sockRequest("GET", endpoint, nil)

	c.Assert(err, check.IsNil)
	c.Assert(status, check.Equals, http.StatusOK)

	var imageJSON types.ImageInspect
	if err = json.Unmarshal(body, &imageJSON); err != nil {
		c.Fatalf("unable to unmarshal body for latest version: %v", err)
	}

	c.Assert(len(imageJSON.RepoTags), check.Equals, 2)

	c.Assert(stringutils.InSlice(imageJSON.RepoTags, "busybox:latest"), check.Equals, true)
	c.Assert(stringutils.InSlice(imageJSON.RepoTags, "busybox:mytag"), check.Equals, true)
}

// #17131, #17139, #17173
func (s *DockerSuite) TestInspectApiEmptyFieldsInConfigPre121(c *check.C) {
	out, _ := dockerCmd(c, "run", "-d", "busybox", "true")

	cleanedContainerID := strings.TrimSpace(out)

	cases := []string{"1.19", "1.20"}
	for _, version := range cases {
		endpoint := fmt.Sprintf("/v%s/containers/%s/json", version, cleanedContainerID)
		status, body, err := sockRequest("GET", endpoint, nil)
		c.Assert(err, check.IsNil)
		c.Assert(status, check.Equals, http.StatusOK)

		var inspectJSON map[string]interface{}
		if err = json.Unmarshal(body, &inspectJSON); err != nil {
			c.Fatalf("unable to unmarshal body for version %s: %v", version, err)
		}

		config, ok := inspectJSON["Config"]
		if !ok {
			c.Fatal("Unable to find 'Config'")
		}
		cfg := config.(map[string]interface{})
		for _, f := range []string{"MacAddress", "NetworkDisabled", "ExposedPorts"} {
			if _, ok := cfg[f]; !ok {
				c.Fatalf("Api version %s expected to include %s in 'Config'", version, f)
			}
		}
	}
}
