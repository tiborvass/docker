package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/tiborvass/docker/api/types"
	registrytypes "github.com/tiborvass/docker/api/types/registry"
	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/net/context"
)

func TestServiceCreateError(t *testing.T) {
	client := &Client{
		client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}
	_, err := client.ServiceCreate(context.Background(), swarm.ServiceSpec{}, types.ServiceCreateOptions{})
	if err == nil || err.Error() != "Error response from daemon: Server error" {
		t.Fatalf("expected a Server Error, got %v", err)
	}
}

func TestServiceCreate(t *testing.T) {
	expectedURL := "/services/create"
	client := &Client{
		client: newMockClient(func(req *http.Request) (*http.Response, error) {
			if !strings.HasPrefix(req.URL.Path, expectedURL) {
				return nil, fmt.Errorf("Expected URL '%s', got '%s'", expectedURL, req.URL)
			}
			if req.Method != "POST" {
				return nil, fmt.Errorf("expected POST method, got %s", req.Method)
			}
			b, err := json.Marshal(types.ServiceCreateResponse{
				ID: "service_id",
			})
			if err != nil {
				return nil, err
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(b)),
			}, nil
		}),
	}

	r, err := client.ServiceCreate(context.Background(), swarm.ServiceSpec{}, types.ServiceCreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "service_id" {
		t.Fatalf("expected `service_id`, got %s", r.ID)
	}
}

func TestServiceCreateCompatiblePlatforms(t *testing.T) {
	var platforms []v1.Platform
	client := &Client{
		client: newMockClient(func(req *http.Request) (*http.Response, error) {
			if strings.HasPrefix(req.URL.Path, "/services/create") {
				// platforms should have been resolved by now
				if len(platforms) != 1 || platforms[0].Architecture != "amd64" || platforms[0].OS != "linux" {
					return nil, fmt.Errorf("incorrect platform information")
				}
				b, err := json.Marshal(types.ServiceCreateResponse{
					ID: "service_" + platforms[0].Architecture,
				})
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader(b)),
				}, nil
			} else if strings.HasPrefix(req.URL.Path, "/distribution/") {
				platforms = []v1.Platform{
					{
						Architecture: "amd64",
						OS:           "linux",
					},
				}
				b, err := json.Marshal(registrytypes.DistributionInspect{
					Descriptor: v1.Descriptor{},
					Platforms:  platforms,
				})
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader(b)),
				}, nil
			} else {
				return nil, fmt.Errorf("unexpected URL '%s'", req.URL.Path)
			}
		}),
	}

	r, err := client.ServiceCreate(context.Background(), swarm.ServiceSpec{}, types.ServiceCreateOptions{QueryRegistry: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "service_amd64" {
		t.Fatalf("expected `service_amd64`, got %s", r.ID)
	}
}
