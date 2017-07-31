package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-check/check"
	"github.com/moby/moby-core/api/types"
	"github.com/moby/moby-core/dockerversion"
	"github.com/moby/moby-core/integration-cli/checker"
	"github.com/moby/moby-core/integration-cli/request"
)

func (s *DockerSuite) TestGetVersion(c *check.C) {
	status, body, err := request.SockRequest("GET", "/version", nil, daemonHost())
	c.Assert(status, checker.Equals, http.StatusOK)
	c.Assert(err, checker.IsNil)

	var v types.Version

	c.Assert(json.Unmarshal(body, &v), checker.IsNil)

	c.Assert(v.Version, checker.Equals, dockerversion.Version, check.Commentf("Version mismatch"))
}
