// +build !windows

package authz

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	eventtypes "github.com/tiborvass/docker/api/types/events"
	"github.com/tiborvass/docker/integration/internal/api/container"
	"github.com/tiborvass/docker/integration/internal/api/image"
	"github.com/tiborvass/docker/integration/internal/api/system"
	"github.com/tiborvass/docker/integration/util/request"
	"github.com/tiborvass/docker/pkg/authorization"
	"github.com/gotestyourself/gotestyourself/skip"
	"github.com/stretchr/testify/require"
)

const (
	testAuthZPlugin     = "authzplugin"
	unauthorizedMessage = "User unauthorized authz plugin"
	errorMessage        = "something went wrong..."
	serverVersionAPI    = "/version"
)

var (
	alwaysAllowed = []string{"/_ping", "/info"}
	ctrl          *authorizationController
)

type authorizationController struct {
	reqRes          authorization.Response // reqRes holds the plugin response to the initial client request
	resRes          authorization.Response // resRes holds the plugin response to the daemon response
	versionReqCount int                    // versionReqCount counts the number of requests to the server version API endpoint
	versionResCount int                    // versionResCount counts the number of responses from the server version API endpoint
	requestsURIs    []string               // requestsURIs stores all request URIs that are sent to the authorization controller
	reqUser         string
	resUser         string
}

func setupTestV1(t *testing.T) func() {
	ctrl = &authorizationController{}
	teardown := setupTest(t)

	err := os.MkdirAll("/etc/docker/plugins", 0755)
	require.Nil(t, err)

	fileName := fmt.Sprintf("/etc/docker/plugins/%s.spec", testAuthZPlugin)
	err = ioutil.WriteFile(fileName, []byte(server.URL), 0644)
	require.Nil(t, err)

	return func() {
		err := os.RemoveAll("/etc/docker/plugins")
		require.Nil(t, err)

		teardown()
		ctrl = nil
	}
}

// check for always allowed endpoints to not inhibit test framework functions
func isAllowed(reqURI string) bool {
	for _, endpoint := range alwaysAllowed {
		if strings.HasSuffix(reqURI, endpoint) {
			return true
		}
	}
	return false
}

func TestAuthZPluginAllowRequest(t *testing.T) {
	defer setupTestV1(t)()
	ctrl.reqRes.Allow = true
	ctrl.resRes.Allow = true
	d.StartWithBusybox(t, "--authorization-plugin="+testAuthZPlugin)

	client, err := d.NewClient()
	require.Nil(t, err)

	// Ensure command successful
	id, err := container.Run(client, "busybox", []string{"top"})
	require.Nil(t, err)

	assertURIRecorded(t, ctrl.requestsURIs, "/containers/create")
	assertURIRecorded(t, ctrl.requestsURIs, fmt.Sprintf("/containers/%s/start", id))

	_, err = system.Version(client)
	require.Nil(t, err)
	require.Equal(t, 1, ctrl.versionReqCount)
	require.Equal(t, 1, ctrl.versionResCount)
}

func TestAuthZPluginTLS(t *testing.T) {
	defer setupTestV1(t)()
	const (
		testDaemonHTTPSAddr = "tcp://localhost:4271"
		cacertPath          = "../../testdata/https/ca.pem"
		serverCertPath      = "../../testdata/https/server-cert.pem"
		serverKeyPath       = "../../testdata/https/server-key.pem"
		clientCertPath      = "../../testdata/https/client-cert.pem"
		clientKeyPath       = "../../testdata/https/client-key.pem"
	)

	d.Start(t,
		"--authorization-plugin="+testAuthZPlugin,
		"--tlsverify",
		"--tlscacert", cacertPath,
		"--tlscert", serverCertPath,
		"--tlskey", serverKeyPath,
		"-H", testDaemonHTTPSAddr)

	ctrl.reqRes.Allow = true
	ctrl.resRes.Allow = true

	client, err := request.NewTLSAPIClient(t, testDaemonHTTPSAddr, cacertPath, clientCertPath, clientKeyPath)
	require.Nil(t, err)

	_, err = system.Version(client)
	require.Nil(t, err)

	require.Equal(t, "client", ctrl.reqUser)
	require.Equal(t, "client", ctrl.resUser)
}

func TestAuthZPluginDenyRequest(t *testing.T) {
	defer setupTestV1(t)()
	d.Start(t, "--authorization-plugin="+testAuthZPlugin)
	ctrl.reqRes.Allow = false
	ctrl.reqRes.Msg = unauthorizedMessage

	client, err := d.NewClient()
	require.Nil(t, err)

	// Ensure command is blocked
	_, err = system.Version(client)
	require.NotNil(t, err)
	require.Equal(t, 1, ctrl.versionReqCount)
	require.Equal(t, 0, ctrl.versionResCount)

	// Ensure unauthorized message appears in response
	require.Equal(t, fmt.Sprintf("Error response from daemon: authorization denied by plugin %s: %s", testAuthZPlugin, unauthorizedMessage), err.Error())
}

// TestAuthZPluginAPIDenyResponse validates that when authorization
// plugin deny the request, the status code is forbidden
func TestAuthZPluginAPIDenyResponse(t *testing.T) {
	defer setupTestV1(t)()
	d.Start(t, "--authorization-plugin="+testAuthZPlugin)
	ctrl.reqRes.Allow = false
	ctrl.resRes.Msg = unauthorizedMessage

	daemonURL, err := url.Parse(d.Sock())
	require.Nil(t, err)

	conn, err := net.DialTimeout(daemonURL.Scheme, daemonURL.Path, time.Second*10)
	require.Nil(t, err)
	client := httputil.NewClientConn(conn, nil)
	req, err := http.NewRequest("GET", "/version", nil)
	require.Nil(t, err)
	resp, err := client.Do(req)

	require.Nil(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAuthZPluginDenyResponse(t *testing.T) {
	defer setupTestV1(t)()
	d.Start(t, "--authorization-plugin="+testAuthZPlugin)
	ctrl.reqRes.Allow = true
	ctrl.resRes.Allow = false
	ctrl.resRes.Msg = unauthorizedMessage

	client, err := d.NewClient()
	require.Nil(t, err)

	// Ensure command is blocked
	_, err = system.Version(client)
	require.NotNil(t, err)
	require.Equal(t, 1, ctrl.versionReqCount)
	require.Equal(t, 1, ctrl.versionResCount)

	// Ensure unauthorized message appears in response
	require.Equal(t, fmt.Sprintf("Error response from daemon: authorization denied by plugin %s: %s", testAuthZPlugin, unauthorizedMessage), err.Error())
}

// TestAuthZPluginAllowEventStream verifies event stream propagates
// correctly after request pass through by the authorization plugin
func TestAuthZPluginAllowEventStream(t *testing.T) {
	skip.IfCondition(t, testEnv.DaemonInfo.OSType != "linux")

	defer setupTestV1(t)()
	ctrl.reqRes.Allow = true
	ctrl.resRes.Allow = true
	d.StartWithBusybox(t, "--authorization-plugin="+testAuthZPlugin)

	client, err := d.NewClient()
	require.Nil(t, err)

	startTime := strconv.FormatInt(system.Time(t, client, testEnv).Unix(), 10)
	events, errs, cancel := system.EventsSince(client, startTime)
	defer cancel()

	// Create a container and wait for the creation events
	id, err := container.Run(client, "busybox", []string{"top"})
	require.Nil(t, err)
	for i := 0; i < 100; i++ {
		c, err := client.ContainerInspect(context.Background(), id)
		require.Nil(t, err)
		if c.State.Running {
			break
		}
		if i == 99 {
			t.Fatal("Container didn't run within 10s")
		}
		time.Sleep(100 * time.Millisecond)
	}

	created := false
	started := false
	for !created && !started {
		select {
		case event := <-events:
			if event.Type == eventtypes.ContainerEventType && event.Actor.ID == id {
				if event.Action == "create" {
					created = true
				}
				if event.Action == "start" {
					started = true
				}
			}
		case err := <-errs:
			if err == io.EOF {
				t.Fatal("premature end of event stream")
			}
			require.Nil(t, err)
		case <-time.After(30 * time.Second):
			// Fail the test
			t.Fatal("event stream timeout")
		}
	}

	// Ensure both events and container endpoints are passed to the
	// authorization plugin
	assertURIRecorded(t, ctrl.requestsURIs, "/events")
	assertURIRecorded(t, ctrl.requestsURIs, "/containers/create")
	assertURIRecorded(t, ctrl.requestsURIs, fmt.Sprintf("/containers/%s/start", id))
}

func TestAuthZPluginErrorResponse(t *testing.T) {
	defer setupTestV1(t)()
	d.Start(t, "--authorization-plugin="+testAuthZPlugin)
	ctrl.reqRes.Allow = true
	ctrl.resRes.Err = errorMessage

	client, err := d.NewClient()
	require.Nil(t, err)

	// Ensure command is blocked
	_, err = system.Version(client)
	require.NotNil(t, err)
	require.Equal(t, fmt.Sprintf("Error response from daemon: plugin %s failed with error: %s: %s", testAuthZPlugin, authorization.AuthZApiResponse, errorMessage), err.Error())
}

func TestAuthZPluginErrorRequest(t *testing.T) {
	defer setupTestV1(t)()
	d.Start(t, "--authorization-plugin="+testAuthZPlugin)
	ctrl.reqRes.Err = errorMessage

	client, err := d.NewClient()
	require.Nil(t, err)

	// Ensure command is blocked
	_, err = system.Version(client)
	require.NotNil(t, err)
	require.Equal(t, fmt.Sprintf("Error response from daemon: plugin %s failed with error: %s: %s", testAuthZPlugin, authorization.AuthZApiRequest, errorMessage), err.Error())
}

func TestAuthZPluginEnsureNoDuplicatePluginRegistration(t *testing.T) {
	defer setupTestV1(t)()
	d.Start(t, "--authorization-plugin="+testAuthZPlugin, "--authorization-plugin="+testAuthZPlugin)

	ctrl.reqRes.Allow = true
	ctrl.resRes.Allow = true

	client, err := d.NewClient()
	require.Nil(t, err)

	_, err = system.Version(client)
	require.Nil(t, err)

	// assert plugin is only called once..
	require.Equal(t, 1, ctrl.versionReqCount)
	require.Equal(t, 1, ctrl.versionResCount)
}

func TestAuthZPluginEnsureLoadImportWorking(t *testing.T) {
	defer setupTestV1(t)()
	ctrl.reqRes.Allow = true
	ctrl.resRes.Allow = true
	d.StartWithBusybox(t, "--authorization-plugin="+testAuthZPlugin, "--authorization-plugin="+testAuthZPlugin)

	client, err := d.NewClient()
	require.Nil(t, err)

	tmp, err := ioutil.TempDir("", "test-authz-load-import")
	require.Nil(t, err)
	defer os.RemoveAll(tmp)

	savedImagePath := filepath.Join(tmp, "save.tar")

	err = image.Save(client, savedImagePath, "busybox")
	require.Nil(t, err)
	err = image.Load(client, savedImagePath)
	require.Nil(t, err)

	exportedImagePath := filepath.Join(tmp, "export.tar")

	id, err := container.Run(client, "busybox", []string{})
	require.Nil(t, err)
	err = container.Export(client, exportedImagePath, id)
	require.Nil(t, err)
	err = image.Import(client, exportedImagePath)
	require.Nil(t, err)
}

func TestAuthZPluginHeader(t *testing.T) {
	defer setupTestV1(t)()
	ctrl.reqRes.Allow = true
	ctrl.resRes.Allow = true
	d.StartWithBusybox(t, "--debug", "--authorization-plugin="+testAuthZPlugin)

	daemonURL, err := url.Parse(d.Sock())
	require.Nil(t, err)

	conn, err := net.DialTimeout(daemonURL.Scheme, daemonURL.Path, time.Second*10)
	require.Nil(t, err)
	client := httputil.NewClientConn(conn, nil)
	req, err := http.NewRequest("GET", "/version", nil)
	require.Nil(t, err)
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, "application/json", resp.Header["Content-Type"][0])
}

// assertURIRecorded verifies that the given URI was sent and recorded
// in the authz plugin
func assertURIRecorded(t *testing.T, uris []string, uri string) {
	var found bool
	for _, u := range uris {
		if strings.Contains(u, uri) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected to find URI '%s', recorded uris '%s'", uri, strings.Join(uris, ","))
	}
}
