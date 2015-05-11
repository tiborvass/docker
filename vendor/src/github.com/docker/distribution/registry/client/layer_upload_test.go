package client

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/testutil"
)

// Test implements distribution.LayerUpload
var _ distribution.LayerUpload = &httpLayerUpload{}

func TestUploadReadFrom(t *testing.T) {
	_, b := newRandomBlob(64)
	repo := "test/upload/readfrom"
	locationPath := fmt.Sprintf("/v2/%s/uploads/testid", repo)

	m := testutil.RequestResponseMap([]testutil.RequestResponseMapping{
		{
			Request: testutil.Request{
				Method: "GET",
				Route:  "/v2/",
			},
			Response: testutil.Response{
				StatusCode: http.StatusOK,
				Headers: http.Header(map[string][]string{
					"Docker-Distribution-API-Version": {"registry/2.0"},
				}),
			},
		},
		// Test Valid case
		{
			Request: testutil.Request{
				Method: "PATCH",
				Route:  locationPath,
				Body:   b,
			},
			Response: testutil.Response{
				StatusCode: http.StatusAccepted,
				Headers: http.Header(map[string][]string{
					"Docker-Upload-UUID": {"46603072-7a1b-4b41-98f9-fd8a7da89f9b"},
					"Location":           {locationPath},
					"Range":              {"0-63"},
				}),
			},
		},
		// Test invalid range
		{
			Request: testutil.Request{
				Method: "PATCH",
				Route:  locationPath,
				Body:   b,
			},
			Response: testutil.Response{
				StatusCode: http.StatusAccepted,
				Headers: http.Header(map[string][]string{
					"Docker-Upload-UUID": {"46603072-7a1b-4b41-98f9-fd8a7da89f9b"},
					"Location":           {locationPath},
					"Range":              {""},
				}),
			},
		},
		// Test 404
		{
			Request: testutil.Request{
				Method: "PATCH",
				Route:  locationPath,
				Body:   b,
			},
			Response: testutil.Response{
				StatusCode: http.StatusNotFound,
			},
		},
		// Test 400 valid json
		{
			Request: testutil.Request{
				Method: "PATCH",
				Route:  locationPath,
				Body:   b,
			},
			Response: testutil.Response{
				StatusCode: http.StatusBadRequest,
				Body: []byte(`
				{
					"errors": [
						{
							"code": "BLOB_UPLOAD_INVALID",
							"message": "invalid upload identifier",
							"detail": "more detail"
						}
					]
				}`),
			},
		},
		// Test 400 invalid json
		{
			Request: testutil.Request{
				Method: "PATCH",
				Route:  locationPath,
				Body:   b,
			},
			Response: testutil.Response{
				StatusCode: http.StatusBadRequest,
				Body:       []byte("something bad happened"),
			},
		},
		// Test 500
		{
			Request: testutil.Request{
				Method: "PATCH",
				Route:  locationPath,
				Body:   b,
			},
			Response: testutil.Response{
				StatusCode: http.StatusInternalServerError,
			},
		},
	})

	e, c := testServer(m)
	defer c()

	repoConfig := &RepositoryConfig{}
	client, err := repoConfig.HTTPClient()
	if err != nil {
		t.Fatalf("Error creating client: %s", err)
	}
	layerUpload := &httpLayerUpload{
		client: client,
	}

	// Valid case
	layerUpload.location = e + locationPath
	n, err := layerUpload.ReadFrom(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("Error calling ReadFrom: %s", err)
	}
	if n != 64 {
		t.Fatalf("Wrong length returned from ReadFrom: %d, expected 64", n)
	}

	// Bad range
	layerUpload.location = e + locationPath
	_, err = layerUpload.ReadFrom(bytes.NewReader(b))
	if err == nil {
		t.Fatalf("Expected error when bad range received")
	}

	// 404
	layerUpload.location = e + locationPath
	_, err = layerUpload.ReadFrom(bytes.NewReader(b))
	if err == nil {
		t.Fatalf("Expected error when not found")
	}
	if blobErr, ok := err.(*BlobUploadNotFoundError); !ok {
		t.Fatalf("Wrong error type %T: %s", err, err)
	} else if expected := e + locationPath; blobErr.Location != expected {
		t.Fatalf("Unexpected location: %s, expected %s", blobErr.Location, expected)
	}

	// 400 valid json
	layerUpload.location = e + locationPath
	_, err = layerUpload.ReadFrom(bytes.NewReader(b))
	if err == nil {
		t.Fatalf("Expected error when not found")
	}
	if uploadErr, ok := err.(*v2.Errors); !ok {
		t.Fatalf("Wrong error type %T: %s", err, err)
	} else if len(uploadErr.Errors) != 1 {
		t.Fatalf("Unexpected number of errors: %d, expected 1", len(uploadErr.Errors))
	} else {
		v2Err := uploadErr.Errors[0]
		if v2Err.Code != v2.ErrorCodeBlobUploadInvalid {
			t.Fatalf("Unexpected error code: %s, expected %s", v2Err.Code.String(), v2.ErrorCodeBlobUploadInvalid.String())
		}
		if expected := "invalid upload identifier"; v2Err.Message != expected {
			t.Fatalf("Unexpected error message: %s, expected %s", v2Err.Message, expected)
		}
		if expected := "more detail"; v2Err.Detail.(string) != expected {
			t.Fatalf("Unexpected error message: %s, expected %s", v2Err.Detail.(string), expected)
		}
	}

	// 400 invalid json
	layerUpload.location = e + locationPath
	_, err = layerUpload.ReadFrom(bytes.NewReader(b))
	if err == nil {
		t.Fatalf("Expected error when not found")
	}
	if uploadErr, ok := err.(*UnexpectedHTTPResponseError); !ok {
		t.Fatalf("Wrong error type %T: %s", err, err)
	} else {
		respStr := string(uploadErr.Response)
		if expected := "something bad happened"; respStr != expected {
			t.Fatalf("Unexpected response string: %s, expected: %s", respStr, expected)
		}
	}

	// 500
	layerUpload.location = e + locationPath
	_, err = layerUpload.ReadFrom(bytes.NewReader(b))
	if err == nil {
		t.Fatalf("Expected error when not found")
	}
	if uploadErr, ok := err.(*UnexpectedHTTPStatusError); !ok {
		t.Fatalf("Wrong error type %T: %s", err, err)
	} else if expected := "500 " + http.StatusText(http.StatusInternalServerError); uploadErr.Status != expected {
		t.Fatalf("Unexpected response status: %s, expected %s", uploadErr.Status, expected)
	}
}

//repo   distribution.Repository
//client *http.Client

//uuid      string
//startedAt time.Time

//location string // always the last value of the location header.
//offset   int64
//closed   bool
