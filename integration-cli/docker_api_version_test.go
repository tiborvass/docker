package main

import (
	"encoding/json"
	"net/http"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/autogen/dockerversion"
	"github.com/go-check/check"
)

func (s *DockerSuite) TestGetVersion(c *check.C) {
	status, body, err := sockRequest("GET", "/version", nil)
	c.Assert(status, check.Equals, http.StatusOK)
	c.Assert(err, check.IsNil)

	var v types.Version
	if err := json.Unmarshal(body, &v); err != nil {
		c.Fatal(err)
	}

	if v.Version != dockerversion.VERSION {
		c.Fatal("Version mismatch")
	}
}
