package main

import (
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/tiborvass/docker/api"
	"github.com/tiborvass/docker/pkg/integration/checker"
	"github.com/go-check/check"
)

func (s *DockerSuite) TestApiOptionsRoute(c *check.C) {
	status, _, err := sockRequest("OPTIONS", "/", nil)
	c.Assert(err, checker.IsNil)
	c.Assert(status, checker.Equals, http.StatusOK)
}

func (s *DockerSuite) TestApiGetEnabledCors(c *check.C) {
	res, body, err := sockRequestRaw("GET", "/version", nil, "")
	c.Assert(err, checker.IsNil)
	c.Assert(res.StatusCode, checker.Equals, http.StatusOK)
	body.Close()
	// TODO: @runcom incomplete tests, why old integration tests had this headers
	// and here none of the headers below are in the response?
	//c.Log(res.Header)
	//c.Assert(res.Header.Get("Access-Control-Allow-Origin"), check.Equals, "*")
	//c.Assert(res.Header.Get("Access-Control-Allow-Headers"), check.Equals, "Origin, X-Requested-With, Content-Type, Accept, X-Registry-Auth")
}

func (s *DockerSuite) TestApiVersionStatusCode(c *check.C) {
	conn, err := sockConn(time.Duration(10 * time.Second))
	c.Assert(err, checker.IsNil)

	client := httputil.NewClientConn(conn, nil)
	defer client.Close()

	req, err := http.NewRequest("GET", "/v999.0/version", nil)
	c.Assert(err, checker.IsNil)
	req.Header.Set("User-Agent", "Docker-Client/999.0 (os)")

	res, err := client.Do(req)
	c.Assert(res.StatusCode, checker.Equals, http.StatusBadRequest)
}

func (s *DockerSuite) TestApiClientVersionNewerThanServer(c *check.C) {
	v := strings.Split(string(api.Version), ".")
	vMinInt, err := strconv.Atoi(v[1])
	c.Assert(err, checker.IsNil)
	vMinInt++
	v[1] = strconv.Itoa(vMinInt)
	version := strings.Join(v, ".")

	status, body, err := sockRequest("GET", "/v"+version+"/version", nil)
	c.Assert(err, checker.IsNil)
	c.Assert(status, checker.Equals, http.StatusBadRequest)
	c.Assert(len(string(body)), check.Not(checker.Equals), 0) // Expected not empty body
}

func (s *DockerSuite) TestApiClientVersionOldNotSupported(c *check.C) {
	v := strings.Split(string(api.MinVersion), ".")
	vMinInt, err := strconv.Atoi(v[1])
	c.Assert(err, checker.IsNil)
	vMinInt--
	v[1] = strconv.Itoa(vMinInt)
	version := strings.Join(v, ".")

	status, body, err := sockRequest("GET", "/v"+version+"/version", nil)
	c.Assert(err, checker.IsNil)
	c.Assert(status, checker.Equals, http.StatusBadRequest)
	c.Assert(len(string(body)), checker.Not(check.Equals), 0) // Expected not empty body
}
