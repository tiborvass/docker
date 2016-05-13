// +build !windows

// TODO Windows: This uses a Unix socket for testing. This might be possible
// to port to Windows using a named pipe instead.

package authorization

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"reflect"
	"testing"

	"bytes"
	"strings"

	"github.com/tiborvass/docker/pkg/plugins"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/gorilla/mux"
)

const pluginAddress = "authzplugin.sock"

func TestAuthZRequestPluginError(t *testing.T) {
	server := authZPluginTestServer{t: t}
	go server.start()
	defer server.stop()

	authZPlugin := createTestPlugin(t)

	request := Request{
		User:           "user",
		RequestBody:    []byte("sample body"),
		RequestURI:     "www.authz.com",
		RequestMethod:  "GET",
		RequestHeaders: map[string]string{"header": "value"},
	}
	server.replayResponse = Response{
		Err: "an error",
	}

	actualResponse, err := authZPlugin.AuthZRequest(&request)
	if err != nil {
		t.Fatalf("Failed to authorize request %v", err)
	}

	if !reflect.DeepEqual(server.replayResponse, *actualResponse) {
		t.Fatalf("Response must be equal")
	}
	if !reflect.DeepEqual(request, server.recordedRequest) {
		t.Fatalf("Requests must be equal")
	}
}

func TestAuthZRequestPlugin(t *testing.T) {
	server := authZPluginTestServer{t: t}
	go server.start()
	defer server.stop()

	authZPlugin := createTestPlugin(t)

	request := Request{
		User:           "user",
		RequestBody:    []byte("sample body"),
		RequestURI:     "www.authz.com",
		RequestMethod:  "GET",
		RequestHeaders: map[string]string{"header": "value"},
	}
	server.replayResponse = Response{
		Allow: true,
		Msg:   "Sample message",
	}

	actualResponse, err := authZPlugin.AuthZRequest(&request)
	if err != nil {
		t.Fatalf("Failed to authorize request %v", err)
	}

	if !reflect.DeepEqual(server.replayResponse, *actualResponse) {
		t.Fatalf("Response must be equal")
	}
	if !reflect.DeepEqual(request, server.recordedRequest) {
		t.Fatalf("Requests must be equal")
	}
}

func TestAuthZResponsePlugin(t *testing.T) {
	server := authZPluginTestServer{t: t}
	go server.start()
	defer server.stop()

	authZPlugin := createTestPlugin(t)

	request := Request{
		User:        "user",
		RequestBody: []byte("sample body"),
	}
	server.replayResponse = Response{
		Allow: true,
		Msg:   "Sample message",
	}

	actualResponse, err := authZPlugin.AuthZResponse(&request)
	if err != nil {
		t.Fatalf("Failed to authorize request %v", err)
	}

	if !reflect.DeepEqual(server.replayResponse, *actualResponse) {
		t.Fatalf("Response must be equal")
	}
	if !reflect.DeepEqual(request, server.recordedRequest) {
		t.Fatalf("Requests must be equal")
	}
}

func TestResponseModifier(t *testing.T) {
	r := httptest.NewRecorder()
	m := NewResponseModifier(r)
	m.Header().Set("h1", "v1")
	m.Write([]byte("body"))
	m.WriteHeader(500)

	m.FlushAll()
	if r.Header().Get("h1") != "v1" {
		t.Fatalf("Header value must exists %s", r.Header().Get("h1"))
	}
	if !reflect.DeepEqual(r.Body.Bytes(), []byte("body")) {
		t.Fatalf("Body value must exists %s", r.Body.Bytes())
	}
	if r.Code != 500 {
		t.Fatalf("Status code must be correct %d", r.Code)
	}
}

func TestDrainBody(t *testing.T) {
	tests := []struct {
		length             int // length is the message length send to drainBody
		expectedBodyLength int // expectedBodyLength is the expected body length after drainBody is called
	}{
		{10, 10}, // Small message size
		{maxBodySize - 1, maxBodySize - 1}, // Max message size
		{maxBodySize * 2, 0},               // Large message size (skip copying body)

	}

	for _, test := range tests {
		msg := strings.Repeat("a", test.length)
		body, closer, err := drainBody(ioutil.NopCloser(bytes.NewReader([]byte(msg))))
		if err != nil {
			t.Fatal(err)
		}
		if len(body) != test.expectedBodyLength {
			t.Fatalf("Body must be copied, actual length: '%d'", len(body))
		}
		if closer == nil {
			t.Fatalf("Closer must not be nil")
		}
		modified, err := ioutil.ReadAll(closer)
		if err != nil {
			t.Fatalf("Error must not be nil: '%v'", err)
		}
		if len(modified) != len(msg) {
			t.Fatalf("Result should not be truncated. Original length: '%d', new length: '%d'", len(msg), len(modified))
		}
	}
}

func TestResponseModifierOverride(t *testing.T) {
	r := httptest.NewRecorder()
	m := NewResponseModifier(r)
	m.Header().Set("h1", "v1")
	m.Write([]byte("body"))
	m.WriteHeader(500)

	overrideHeader := make(http.Header)
	overrideHeader.Add("h1", "v2")
	overrideHeaderBytes, err := json.Marshal(overrideHeader)
	if err != nil {
		t.Fatalf("override header failed %v", err)
	}

	m.OverrideHeader(overrideHeaderBytes)
	m.OverrideBody([]byte("override body"))
	m.OverrideStatusCode(404)
	m.FlushAll()
	if r.Header().Get("h1") != "v2" {
		t.Fatalf("Header value must exists %s", r.Header().Get("h1"))
	}
	if !reflect.DeepEqual(r.Body.Bytes(), []byte("override body")) {
		t.Fatalf("Body value must exists %s", r.Body.Bytes())
	}
	if r.Code != 404 {
		t.Fatalf("Status code must be correct %d", r.Code)
	}
}

// createTestPlugin creates a new sample authorization plugin
func createTestPlugin(t *testing.T) *authorizationPlugin {
	plugin := &plugins.Plugin{Name: "authz"}
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	plugin.Client, err = plugins.NewClient("unix:///"+path.Join(pwd, pluginAddress), tlsconfig.Options{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("Failed to create client %v", err)
	}

	return &authorizationPlugin{name: "plugin", plugin: plugin}
}

// AuthZPluginTestServer is a simple server that implements the authZ plugin interface
type authZPluginTestServer struct {
	listener net.Listener
	t        *testing.T
	// request stores the request sent from the daemon to the plugin
	recordedRequest Request
	// response stores the response sent from the plugin to the daemon
	replayResponse Response
}

// start starts the test server that implements the plugin
func (t *authZPluginTestServer) start() {
	r := mux.NewRouter()
	os.Remove(pluginAddress)
	l, _ := net.ListenUnix("unix", &net.UnixAddr{Name: pluginAddress, Net: "unix"})
	t.listener = l
	r.HandleFunc("/Plugin.Activate", t.activate)
	r.HandleFunc("/"+AuthZApiRequest, t.auth)
	r.HandleFunc("/"+AuthZApiResponse, t.auth)
	server := http.Server{Handler: r, Addr: pluginAddress}
	server.Serve(l)
}

// stop stops the test server that implements the plugin
func (t *authZPluginTestServer) stop() {
	os.Remove(pluginAddress)
	if t.listener != nil {
		t.listener.Close()
	}
}

// auth is a used to record/replay the authentication api messages
func (t *authZPluginTestServer) auth(w http.ResponseWriter, r *http.Request) {
	t.recordedRequest = Request{}
	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	json.Unmarshal(body, &t.recordedRequest)
	b, _ := json.Marshal(t.replayResponse)
	w.Write(b)
}

func (t *authZPluginTestServer) activate(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(plugins.Manifest{Implements: []string{AuthZApiImplements}})
	w.Write(b)
}
