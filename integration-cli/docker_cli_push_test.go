package main

import (
	"archive/tar"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/integration-cli/checker"
	"github.com/tiborvass/docker/integration-cli/cli/build"
	"github.com/go-check/check"
	"github.com/gotestyourself/gotestyourself/icmd"
)

// Pushing an image to a private registry.
func testPushBusyboxImage(c *check.C) {
	repoName := fmt.Sprintf("%v/dockercli/busybox", privateRegistryURL)
	// tag the image to upload it to the private registry
	dockerCmd(c, "tag", "busybox", repoName)
	// push the image to the registry
	dockerCmd(c, "push", repoName)
}

func (s *DockerRegistrySuite) TestPushBusyboxImage(c *check.C) {
	testPushBusyboxImage(c)
}

func (s *DockerSchema1RegistrySuite) TestPushBusyboxImage(c *check.C) {
	testPushBusyboxImage(c)
}

// pushing an image without a prefix should throw an error
func (s *DockerSuite) TestPushUnprefixedRepo(c *check.C) {
	out, _, err := dockerCmdWithError("push", "busybox")
	c.Assert(err, check.NotNil, check.Commentf("pushing an unprefixed repo didn't result in a non-zero exit status: %s", out))
}

func testPushUntagged(c *check.C) {
	repoName := fmt.Sprintf("%v/dockercli/busybox", privateRegistryURL)
	expected := "An image does not exist locally with the tag"

	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf("pushing the image to the private registry should have failed: output %q", out))
	c.Assert(out, checker.Contains, expected, check.Commentf("pushing the image failed"))
}

func (s *DockerRegistrySuite) TestPushUntagged(c *check.C) {
	testPushUntagged(c)
}

func (s *DockerSchema1RegistrySuite) TestPushUntagged(c *check.C) {
	testPushUntagged(c)
}

func testPushBadTag(c *check.C) {
	repoName := fmt.Sprintf("%v/dockercli/busybox:latest", privateRegistryURL)
	expected := "does not exist"

	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf("pushing the image to the private registry should have failed: output %q", out))
	c.Assert(out, checker.Contains, expected, check.Commentf("pushing the image failed"))
}

func (s *DockerRegistrySuite) TestPushBadTag(c *check.C) {
	testPushBadTag(c)
}

func (s *DockerSchema1RegistrySuite) TestPushBadTag(c *check.C) {
	testPushBadTag(c)
}

func testPushMultipleTags(c *check.C) {
	repoName := fmt.Sprintf("%v/dockercli/busybox", privateRegistryURL)
	repoTag1 := fmt.Sprintf("%v/dockercli/busybox:t1", privateRegistryURL)
	repoTag2 := fmt.Sprintf("%v/dockercli/busybox:t2", privateRegistryURL)
	// tag the image and upload it to the private registry
	dockerCmd(c, "tag", "busybox", repoTag1)

	dockerCmd(c, "tag", "busybox", repoTag2)

	dockerCmd(c, "push", repoName)

	// Ensure layer list is equivalent for repoTag1 and repoTag2
	out1, _ := dockerCmd(c, "pull", repoTag1)

	imageAlreadyExists := ": Image already exists"
	var out1Lines []string
	for _, outputLine := range strings.Split(out1, "\n") {
		if strings.Contains(outputLine, imageAlreadyExists) {
			out1Lines = append(out1Lines, outputLine)
		}
	}

	out2, _ := dockerCmd(c, "pull", repoTag2)

	var out2Lines []string
	for _, outputLine := range strings.Split(out2, "\n") {
		if strings.Contains(outputLine, imageAlreadyExists) {
			out1Lines = append(out1Lines, outputLine)
		}
	}
	c.Assert(out2Lines, checker.HasLen, len(out1Lines))

	for i := range out1Lines {
		c.Assert(out1Lines[i], checker.Equals, out2Lines[i])
	}
}

func (s *DockerRegistrySuite) TestPushMultipleTags(c *check.C) {
	testPushMultipleTags(c)
}

func (s *DockerSchema1RegistrySuite) TestPushMultipleTags(c *check.C) {
	testPushMultipleTags(c)
}

func testPushEmptyLayer(c *check.C) {
	repoName := fmt.Sprintf("%v/dockercli/emptylayer", privateRegistryURL)
	emptyTarball, err := ioutil.TempFile("", "empty_tarball")
	c.Assert(err, check.IsNil, check.Commentf("Unable to create test file"))

	tw := tar.NewWriter(emptyTarball)
	err = tw.Close()
	c.Assert(err, check.IsNil, check.Commentf("Error creating empty tarball"))

	freader, err := os.Open(emptyTarball.Name())
	c.Assert(err, check.IsNil, check.Commentf("Could not open test tarball"))
	defer freader.Close()

	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerBinary, "import", "-", repoName},
		Stdin:   freader,
	}).Assert(c, icmd.Success)

	// Now verify we can push it
	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.IsNil, check.Commentf("pushing the image to the private registry has failed: %s", out))
}

func (s *DockerRegistrySuite) TestPushEmptyLayer(c *check.C) {
	testPushEmptyLayer(c)
}

func (s *DockerSchema1RegistrySuite) TestPushEmptyLayer(c *check.C) {
	testPushEmptyLayer(c)
}

// testConcurrentPush pushes multiple tags to the same repo
// concurrently.
func testConcurrentPush(c *check.C) {
	repoName := fmt.Sprintf("%v/dockercli/busybox", privateRegistryURL)

	var repos []string
	for _, tag := range []string{"push1", "push2", "push3"} {
		repo := fmt.Sprintf("%v:%v", repoName, tag)
		buildImageSuccessfully(c, repo, build.WithDockerfile(fmt.Sprintf(`
	FROM busybox
	ENTRYPOINT ["/bin/echo"]
	ENV FOO foo
	ENV BAR bar
	CMD echo %s
`, repo)))
		repos = append(repos, repo)
	}

	// Push tags, in parallel
	results := make(chan error)

	for _, repo := range repos {
		go func(repo string) {
			result := icmd.RunCommand(dockerBinary, "push", repo)
			results <- result.Error
		}(repo)
	}

	for range repos {
		err := <-results
		c.Assert(err, checker.IsNil, check.Commentf("concurrent push failed with error: %v", err))
	}

	// Clear local images store.
	args := append([]string{"rmi"}, repos...)
	dockerCmd(c, args...)

	// Re-pull and run individual tags, to make sure pushes succeeded
	for _, repo := range repos {
		dockerCmd(c, "pull", repo)
		dockerCmd(c, "inspect", repo)
		out, _ := dockerCmd(c, "run", "--rm", repo)
		c.Assert(strings.TrimSpace(out), checker.Equals, "/bin/sh -c echo "+repo)
	}
}

func (s *DockerRegistrySuite) TestConcurrentPush(c *check.C) {
	testConcurrentPush(c)
}

func (s *DockerSchema1RegistrySuite) TestConcurrentPush(c *check.C) {
	testConcurrentPush(c)
}

func (s *DockerRegistrySuite) TestCrossRepositoryLayerPush(c *check.C) {
	sourceRepoName := fmt.Sprintf("%v/dockercli/busybox", privateRegistryURL)
	// tag the image to upload it to the private registry
	dockerCmd(c, "tag", "busybox", sourceRepoName)
	// push the image to the registry
	out1, _, err := dockerCmdWithError("push", sourceRepoName)
	c.Assert(err, check.IsNil, check.Commentf("pushing the image to the private registry has failed: %s", out1))
	// ensure that none of the layers were mounted from another repository during push
	c.Assert(strings.Contains(out1, "Mounted from"), check.Equals, false)

	digest1 := reference.DigestRegexp.FindString(out1)
	c.Assert(len(digest1), checker.GreaterThan, 0, check.Commentf("no digest found for pushed manifest"))

	destRepoName := fmt.Sprintf("%v/dockercli/crossrepopush", privateRegistryURL)
	// retag the image to upload the same layers to another repo in the same registry
	dockerCmd(c, "tag", "busybox", destRepoName)
	// push the image to the registry
	out2, _, err := dockerCmdWithError("push", destRepoName)
	c.Assert(err, check.IsNil, check.Commentf("pushing the image to the private registry has failed: %s", out2))
	// ensure that layers were mounted from the first repo during push
	c.Assert(strings.Contains(out2, "Mounted from dockercli/busybox"), check.Equals, true)

	digest2 := reference.DigestRegexp.FindString(out2)
	c.Assert(len(digest2), checker.GreaterThan, 0, check.Commentf("no digest found for pushed manifest"))
	c.Assert(digest1, check.Equals, digest2)

	// ensure that pushing again produces the same digest
	out3, _, err := dockerCmdWithError("push", destRepoName)
	c.Assert(err, check.IsNil, check.Commentf("pushing the image to the private registry has failed: %s", out2))

	digest3 := reference.DigestRegexp.FindString(out3)
	c.Assert(len(digest2), checker.GreaterThan, 0, check.Commentf("no digest found for pushed manifest"))
	c.Assert(digest3, check.Equals, digest2)

	// ensure that we can pull and run the cross-repo-pushed repository
	dockerCmd(c, "rmi", destRepoName)
	dockerCmd(c, "pull", destRepoName)
	out4, _ := dockerCmd(c, "run", destRepoName, "echo", "-n", "hello world")
	c.Assert(out4, check.Equals, "hello world")
}

func (s *DockerSchema1RegistrySuite) TestCrossRepositoryLayerPushNotSupported(c *check.C) {
	sourceRepoName := fmt.Sprintf("%v/dockercli/busybox", privateRegistryURL)
	// tag the image to upload it to the private registry
	dockerCmd(c, "tag", "busybox", sourceRepoName)
	// push the image to the registry
	out1, _, err := dockerCmdWithError("push", sourceRepoName)
	c.Assert(err, check.IsNil, check.Commentf("pushing the image to the private registry has failed: %s", out1))
	// ensure that none of the layers were mounted from another repository during push
	c.Assert(strings.Contains(out1, "Mounted from"), check.Equals, false)

	digest1 := reference.DigestRegexp.FindString(out1)
	c.Assert(len(digest1), checker.GreaterThan, 0, check.Commentf("no digest found for pushed manifest"))

	destRepoName := fmt.Sprintf("%v/dockercli/crossrepopush", privateRegistryURL)
	// retag the image to upload the same layers to another repo in the same registry
	dockerCmd(c, "tag", "busybox", destRepoName)
	// push the image to the registry
	out2, _, err := dockerCmdWithError("push", destRepoName)
	c.Assert(err, check.IsNil, check.Commentf("pushing the image to the private registry has failed: %s", out2))
	// schema1 registry should not support cross-repo layer mounts, so ensure that this does not happen
	c.Assert(strings.Contains(out2, "Mounted from"), check.Equals, false)

	digest2 := reference.DigestRegexp.FindString(out2)
	c.Assert(len(digest2), checker.GreaterThan, 0, check.Commentf("no digest found for pushed manifest"))
	c.Assert(digest1, check.Not(check.Equals), digest2)

	// ensure that we can pull and run the second pushed repository
	dockerCmd(c, "rmi", destRepoName)
	dockerCmd(c, "pull", destRepoName)
	out3, _ := dockerCmd(c, "run", destRepoName, "echo", "-n", "hello world")
	c.Assert(out3, check.Equals, "hello world")
}

func (s *DockerRegistryAuthHtpasswdSuite) TestPushNoCredentialsNoRetry(c *check.C) {
	repoName := fmt.Sprintf("%s/busybox", privateRegistryURL)
	dockerCmd(c, "tag", "busybox", repoName)
	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf(out))
	c.Assert(out, check.Not(checker.Contains), "Retrying")
	c.Assert(out, checker.Contains, "no basic auth credentials")
}

// This may be flaky but it's needed not to regress on unauthorized push, see #21054
func (s *DockerSuite) TestPushToCentralRegistryUnauthorized(c *check.C) {
	testRequires(c, Network)
	repoName := "test/busybox"
	dockerCmd(c, "tag", "busybox", repoName)
	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf(out))
	c.Assert(out, check.Not(checker.Contains), "Retrying")
}

func getTestTokenService(status int, body string, retries int) *httptest.Server {
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if retries > 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"errors":[{"code":"UNAVAILABLE","message":"cannot create token at this time"}]}`))
			retries--
		} else {
			w.WriteHeader(status)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(body))
		}
		mu.Unlock()
	}))
}

func (s *DockerRegistryAuthTokenSuite) TestPushTokenServiceUnauthResponse(c *check.C) {
	ts := getTestTokenService(http.StatusUnauthorized, `{"errors": [{"Code":"UNAUTHORIZED", "message": "a message", "detail": null}]}`, 0)
	defer ts.Close()
	s.setupRegistryWithTokenService(c, ts.URL)
	repoName := fmt.Sprintf("%s/busybox", privateRegistryURL)
	dockerCmd(c, "tag", "busybox", repoName)
	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf(out))
	c.Assert(out, checker.Not(checker.Contains), "Retrying")
	c.Assert(out, checker.Contains, "unauthorized: a message")
}

func (s *DockerRegistryAuthTokenSuite) TestPushMisconfiguredTokenServiceResponseUnauthorized(c *check.C) {
	ts := getTestTokenService(http.StatusUnauthorized, `{"error": "unauthorized"}`, 0)
	defer ts.Close()
	s.setupRegistryWithTokenService(c, ts.URL)
	repoName := fmt.Sprintf("%s/busybox", privateRegistryURL)
	dockerCmd(c, "tag", "busybox", repoName)
	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf(out))
	c.Assert(out, checker.Not(checker.Contains), "Retrying")
	split := strings.Split(out, "\n")
	c.Assert(split[len(split)-2], check.Equals, "unauthorized: authentication required")
}

func (s *DockerRegistryAuthTokenSuite) TestPushMisconfiguredTokenServiceResponseError(c *check.C) {
	ts := getTestTokenService(http.StatusTooManyRequests, `{"errors": [{"code":"TOOMANYREQUESTS","message":"out of tokens"}]}`, 3)
	defer ts.Close()
	s.setupRegistryWithTokenService(c, ts.URL)
	repoName := fmt.Sprintf("%s/busybox", privateRegistryURL)
	dockerCmd(c, "tag", "busybox", repoName)
	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf(out))
	// TODO: isolate test so that it can be guaranteed that the 503 will trigger xfer retries
	//c.Assert(out, checker.Contains, "Retrying")
	//c.Assert(out, checker.Not(checker.Contains), "Retrying in 15")
	split := strings.Split(out, "\n")
	c.Assert(split[len(split)-2], check.Equals, "toomanyrequests: out of tokens")
}

func (s *DockerRegistryAuthTokenSuite) TestPushMisconfiguredTokenServiceResponseUnparsable(c *check.C) {
	ts := getTestTokenService(http.StatusForbidden, `no way`, 0)
	defer ts.Close()
	s.setupRegistryWithTokenService(c, ts.URL)
	repoName := fmt.Sprintf("%s/busybox", privateRegistryURL)
	dockerCmd(c, "tag", "busybox", repoName)
	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf(out))
	c.Assert(out, checker.Not(checker.Contains), "Retrying")
	split := strings.Split(out, "\n")
	c.Assert(split[len(split)-2], checker.Contains, "error parsing HTTP 403 response body: ")
}

func (s *DockerRegistryAuthTokenSuite) TestPushMisconfiguredTokenServiceResponseNoToken(c *check.C) {
	ts := getTestTokenService(http.StatusOK, `{"something": "wrong"}`, 0)
	defer ts.Close()
	s.setupRegistryWithTokenService(c, ts.URL)
	repoName := fmt.Sprintf("%s/busybox", privateRegistryURL)
	dockerCmd(c, "tag", "busybox", repoName)
	out, _, err := dockerCmdWithError("push", repoName)
	c.Assert(err, check.NotNil, check.Commentf(out))
	c.Assert(out, checker.Not(checker.Contains), "Retrying")
	split := strings.Split(out, "\n")
	c.Assert(split[len(split)-2], check.Equals, "authorization server did not include a token in the response")
}
