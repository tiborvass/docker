package client // import "github.com/tiborvass/docker/client"

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/tiborvass/docker/errdefs"
)

func TestSwarmInspectError(t *testing.T) {
	client := &Client{
		client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}

	_, err := client.SwarmInspect(context.Background())
	if err == nil || err.Error() != "Error response from daemon: Server error" {
		t.Fatalf("expected a Server Error, got %v", err)
	}
	if !errdefs.IsSystem(err) {
		t.Fatalf("expected a Server Error, got %T", err)
	}
}

func TestSwarmInspect(t *testing.T) {
	expectedURL := "/swarm"
	client := &Client{
		client: newMockClient(func(req *http.Request) (*http.Response, error) {
			if !strings.HasPrefix(req.URL.Path, expectedURL) {
				return nil, fmt.Errorf("Expected URL '%s', got '%s'", expectedURL, req.URL)
			}
			content, err := json.Marshal(swarm.Swarm{
				ClusterInfo: swarm.ClusterInfo{
					ID: "swarm_id",
				},
			})
			if err != nil {
				return nil, err
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(content)),
			}, nil
		}),
	}

	swarmInspect, err := client.SwarmInspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if swarmInspect.ID != "swarm_id" {
		t.Fatalf("expected `swarm_id`, got %s", swarmInspect.ID)
	}
}
