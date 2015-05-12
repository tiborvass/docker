package client

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/testutil"
)

func testServer(rrm testutil.RequestResponseMap) (string, func()) {
	h := testutil.NewHandler(rrm)
	s := httptest.NewServer(h)
	return s.URL, s.Close
}

func newRandomBlob(size int) (digest.Digest, []byte) {
	b := make([]byte, size)
	if n, err := rand.Read(b); err != nil {
		panic(err)
	} else if n != size {
		panic("unable to read enough bytes")
	}

	dgst, err := digest.FromBytes(b)
	if err != nil {
		panic(err)
	}

	return dgst, b
}

func addTestFetch(repo string, dgst digest.Digest, content []byte, m *testutil.RequestResponseMap) {
	*m = append(*m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "GET",
			Route:  "/v2/" + repo + "/blobs/" + dgst.String(),
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Body:       content,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(len(content))},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})
	*m = append(*m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "HEAD",
			Route:  "/v2/" + repo + "/blobs/" + dgst.String(),
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(len(content))},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})
}

func addPing(m *testutil.RequestResponseMap) {
	*m = append(*m, testutil.RequestResponseMapping{
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
	})
}

func TestBlobFetch(t *testing.T) {
	d1, b1 := newRandomBlob(1024)
	var m testutil.RequestResponseMap
	addTestFetch("test.example.com/repo1", d1, b1, &m)
	addPing(&m)

	e, c := testServer(m)
	defer c()

	ctx := context.Background()
	r, err := NewRepository(ctx, "test.example.com/repo1", e, nil)
	if err != nil {
		t.Fatal(err)
	}
	l := r.Blobs(ctx)

	b, err := l.Get(ctx, d1)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(b, b1) != 0 {
		t.Fatalf("Wrong bytes values fetched: [%d]byte != [%d]byte", len(b), len(b1))
	}

	// TODO(dmcgowan): Test error cases
}

func TestBlobExists(t *testing.T) {
	d1, b1 := newRandomBlob(1024)
	var m testutil.RequestResponseMap
	addTestFetch("test.example.com/repo1", d1, b1, &m)
	addPing(&m)

	e, c := testServer(m)
	defer c()

	ctx := context.Background()
	r, err := NewRepository(ctx, "test.example.com/repo1", e, nil)
	if err != nil {
		t.Fatal(err)
	}
	l := r.Blobs(ctx)

	stat, err := l.Stat(ctx, d1)
	if err != nil {
		t.Fatal(err)
	}

	if stat.Digest != d1 {
		t.Fatalf("Unexpected digest: %s, expected %s", stat.Digest, d1)
	}

	if stat.Length != int64(len(b1)) {
		t.Fatalf("Unexpected length: %d, expected %d", stat.Length, len(b1))
	}

	// TODO(dmcgowan): Test error cases and ErrBlobUnknown case
}

func TestBlobUploadChunked(t *testing.T) {
	dgst, b1 := newRandomBlob(1024)
	var m testutil.RequestResponseMap
	addPing(&m)
	chunks := [][]byte{
		b1[0:256],
		b1[256:512],
		b1[512:513],
		b1[513:1024],
	}
	repo := "test.example.com/uploadrepo"
	uuids := []string{uuid.New()}
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "POST",
			Route:  "/v2/" + repo + "/blobs/uploads/",
		},
		Response: testutil.Response{
			StatusCode: http.StatusAccepted,
			Headers: http.Header(map[string][]string{
				"Content-Length":     {"0"},
				"Location":           {"/v2/" + repo + "/blobs/uploads/" + uuids[0]},
				"Docker-Upload-UUID": {uuids[0]},
				"Range":              {"0-0"},
			}),
		},
	})
	offset := 0
	for i, chunk := range chunks {
		uuids = append(uuids, uuid.New())
		newOffset := offset + len(chunk)
		m = append(m, testutil.RequestResponseMapping{
			Request: testutil.Request{
				Method: "PATCH",
				Route:  "/v2/" + repo + "/blobs/uploads/" + uuids[i],
				Body:   chunk,
			},
			Response: testutil.Response{
				StatusCode: http.StatusAccepted,
				Headers: http.Header(map[string][]string{
					"Content-Length":     {"0"},
					"Location":           {"/v2/" + repo + "/blobs/uploads/" + uuids[i+1]},
					"Docker-Upload-UUID": {uuids[i+1]},
					"Range":              {fmt.Sprintf("%d-%d", offset, newOffset-1)},
				}),
			},
		})
		offset = newOffset
	}
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "PUT",
			Route:  "/v2/" + repo + "/blobs/uploads/" + uuids[len(uuids)-1],
			QueryParams: map[string][]string{
				"digest": {dgst.String()},
			},
		},
		Response: testutil.Response{
			StatusCode: http.StatusCreated,
			Headers: http.Header(map[string][]string{
				"Content-Length":        {"0"},
				"Docker-Content-Digest": {dgst.String()},
				"Content-Range":         {fmt.Sprintf("0-%d", offset-1)},
			}),
		},
	})
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "HEAD",
			Route:  "/v2/" + repo + "/blobs/" + dgst.String(),
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(offset)},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})

	e, c := testServer(m)
	defer c()

	ctx := context.Background()
	r, err := NewRepository(ctx, repo, e, nil)
	if err != nil {
		t.Fatal(err)
	}
	l := r.Blobs(ctx)

	upload, err := l.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if upload.ID() != uuids[0] {
		log.Fatalf("Unexpected UUID %s; expected %s", upload.ID(), uuids[0])
	}

	for _, chunk := range chunks {
		n, err := upload.Write(chunk)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(chunk) {
			t.Fatalf("Unexpected length returned from write: %d; expected: %d", n, len(chunk))
		}
	}

	blob, err := upload.Commit(ctx, distribution.Descriptor{
		Digest: dgst,
		Length: int64(len(b1)),
	})
	if err != nil {
		t.Fatal(err)
	}

	if blob.Length != int64(len(b1)) {
		t.Fatalf("Unexpected blob size: %d; expected: %d", blob.Length, len(b1))
	}
}

func TestBlobUploadMonolithic(t *testing.T) {
	dgst, b1 := newRandomBlob(1024)
	var m testutil.RequestResponseMap
	addPing(&m)
	repo := "test.example.com/uploadrepo"
	uploadID := uuid.New()
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "POST",
			Route:  "/v2/" + repo + "/blobs/uploads/",
		},
		Response: testutil.Response{
			StatusCode: http.StatusAccepted,
			Headers: http.Header(map[string][]string{
				"Content-Length":     {"0"},
				"Location":           {"/v2/" + repo + "/blobs/uploads/" + uploadID},
				"Docker-Upload-UUID": {uploadID},
				"Range":              {"0-0"},
			}),
		},
	})
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "PATCH",
			Route:  "/v2/" + repo + "/blobs/uploads/" + uploadID,
			Body:   b1,
		},
		Response: testutil.Response{
			StatusCode: http.StatusAccepted,
			Headers: http.Header(map[string][]string{
				"Location":              {"/v2/" + repo + "/blobs/uploads/" + uploadID},
				"Docker-Upload-UUID":    {uploadID},
				"Content-Length":        {"0"},
				"Docker-Content-Digest": {dgst.String()},
				"Range":                 {fmt.Sprintf("0-%d", len(b1)-1)},
			}),
		},
	})
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "PUT",
			Route:  "/v2/" + repo + "/blobs/uploads/" + uploadID,
			QueryParams: map[string][]string{
				"digest": {dgst.String()},
			},
		},
		Response: testutil.Response{
			StatusCode: http.StatusCreated,
			Headers: http.Header(map[string][]string{
				"Content-Length":        {"0"},
				"Docker-Content-Digest": {dgst.String()},
				"Content-Range":         {fmt.Sprintf("0-%d", len(b1)-1)},
			}),
		},
	})
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "HEAD",
			Route:  "/v2/" + repo + "/blobs/" + dgst.String(),
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(len(b1))},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})

	e, c := testServer(m)
	defer c()

	ctx := context.Background()
	r, err := NewRepository(ctx, repo, e, nil)
	if err != nil {
		t.Fatal(err)
	}
	l := r.Blobs(ctx)

	upload, err := l.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if upload.ID() != uploadID {
		log.Fatalf("Unexpected UUID %s; expected %s", upload.ID(), uploadID)
	}

	n, err := upload.ReadFrom(bytes.NewReader(b1))
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(b1)) {
		t.Fatalf("Unexpected ReadFrom length: %d; expected: %d", n, len(b1))
	}

	blob, err := upload.Commit(ctx, distribution.Descriptor{
		Digest: dgst,
		Length: int64(len(b1)),
	})
	if err != nil {
		t.Fatal(err)
	}

	if blob.Length != int64(len(b1)) {
		t.Fatalf("Unexpected blob size: %d; expected: %d", blob.Length, len(b1))
	}
}

func newRandomSchema1Manifest(name, tag string, blobCount int) (*manifest.SignedManifest, digest.Digest) {
	blobs := make([]manifest.FSLayer, blobCount)
	history := make([]manifest.History, blobCount)

	for i := 0; i < blobCount; i++ {
		dgst, blob := newRandomBlob((i % 5) * 16)

		blobs[i] = manifest.FSLayer{BlobSum: dgst}
		history[i] = manifest.History{V1Compatibility: fmt.Sprintf("{\"Hex\": \"%x\"}", blob)}
	}

	m := &manifest.SignedManifest{
		Manifest: manifest.Manifest{
			Name:         name,
			Tag:          tag,
			Architecture: "x86",
			FSLayers:     blobs,
			History:      history,
			Versioned: manifest.Versioned{
				SchemaVersion: 1,
			},
		},
	}
	manifestBytes, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	dgst, err := digest.FromBytes(manifestBytes)
	if err != nil {
		panic(err)
	}

	m.Raw = manifestBytes

	return m, dgst
}

func addTestManifest(repo, reference string, content []byte, m *testutil.RequestResponseMap) {
	*m = append(*m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "GET",
			Route:  "/v2/" + repo + "/manifests/" + reference,
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Body:       content,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(len(content))},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})
	*m = append(*m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "HEAD",
			Route:  "/v2/" + repo + "/manifests/" + reference,
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(len(content))},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})

}

func checkEqualManifest(m1, m2 *manifest.SignedManifest) error {
	if m1.Name != m2.Name {
		return fmt.Errorf("name does not match %q != %q", m1.Name, m2.Name)
	}
	if m1.Tag != m2.Tag {
		return fmt.Errorf("tag does not match %q != %q", m1.Tag, m2.Tag)
	}
	if len(m1.FSLayers) != len(m2.FSLayers) {
		return fmt.Errorf("fs blob length does not match %d != %d", len(m1.FSLayers), len(m2.FSLayers))
	}
	for i := range m1.FSLayers {
		if m1.FSLayers[i].BlobSum != m2.FSLayers[i].BlobSum {
			return fmt.Errorf("blobsum does not match %q != %q", m1.FSLayers[i].BlobSum, m2.FSLayers[i].BlobSum)
		}
	}
	if len(m1.History) != len(m2.History) {
		return fmt.Errorf("history length does not match %d != %d", len(m1.History), len(m2.History))
	}
	for i := range m1.History {
		if m1.History[i].V1Compatibility != m2.History[i].V1Compatibility {
			return fmt.Errorf("blobsum does not match %q != %q", m1.History[i].V1Compatibility, m2.History[i].V1Compatibility)
		}
	}
	return nil
}

func TestManifestFetch(t *testing.T) {
	repo := "test.example.com/repo"
	m1, dgst := newRandomSchema1Manifest(repo, "latest", 6)
	var m testutil.RequestResponseMap
	addPing(&m)
	addTestManifest(repo, dgst.String(), m1.Raw, &m)

	e, c := testServer(m)
	defer c()

	r, err := NewRepository(context.Background(), repo, e, nil)
	if err != nil {
		t.Fatal(err)
	}
	ms := r.Manifests()

	ok, err := ms.Exists(dgst)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Manifest does not exist")
	}

	manifest, err := ms.Get(dgst)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkEqualManifest(manifest, m1); err != nil {
		t.Fatal(err)
	}
}

func TestManifestFetchByTag(t *testing.T) {
	repo := "test.example.com/repo/by/tag"
	m1, _ := newRandomSchema1Manifest(repo, "latest", 6)
	var m testutil.RequestResponseMap
	addPing(&m)
	addTestManifest(repo, "latest", m1.Raw, &m)

	e, c := testServer(m)
	defer c()

	r, err := NewRepository(context.Background(), repo, e, nil)
	if err != nil {
		t.Fatal(err)
	}

	ms := r.Manifests()
	ok, err := ms.ExistsByTag("latest")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Manifest does not exist")
	}

	manifest, err := ms.GetByTag("latest")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkEqualManifest(manifest, m1); err != nil {
		t.Fatal(err)
	}
}

func TestManifestDelete(t *testing.T) {
	repo := "test.example.com/repo/delete"
	_, dgst1 := newRandomSchema1Manifest(repo, "latest", 6)
	_, dgst2 := newRandomSchema1Manifest(repo, "latest", 6)
	var m testutil.RequestResponseMap
	addPing(&m)
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "DELETE",
			Route:  "/v2/" + repo + "/manifests/" + dgst1.String(),
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Headers: http.Header(map[string][]string{
				"Content-Length": {"0"},
			}),
		},
	})

	e, c := testServer(m)
	defer c()

	r, err := NewRepository(context.Background(), repo, e, nil)
	if err != nil {
		t.Fatal(err)
	}

	ms := r.Manifests()
	if err := ms.Delete(dgst1); err != nil {
		t.Fatal(err)
	}
	if err := ms.Delete(dgst2); err == nil {
		t.Fatal("Expected error deleting unknown manifest")
	}
	// TODO(dmcgowan): Check for specific unknown error
}

func TestManifestPut(t *testing.T) {
	repo := "test.example.com/repo/delete"
	m1, dgst := newRandomSchema1Manifest(repo, "other", 6)
	var m testutil.RequestResponseMap
	addPing(&m)
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "PUT",
			Route:  "/v2/" + repo + "/manifests/other",
			Body:   m1.Raw,
		},
		Response: testutil.Response{
			StatusCode: http.StatusAccepted,
			Headers: http.Header(map[string][]string{
				"Content-Length":        {"0"},
				"Docker-Content-Digest": {dgst.String()},
			}),
		},
	})

	e, c := testServer(m)
	defer c()

	r, err := NewRepository(context.Background(), repo, e, nil)
	if err != nil {
		t.Fatal(err)
	}

	ms := r.Manifests()
	if err := ms.Put(m1); err != nil {
		t.Fatal(err)
	}

	// TODO(dmcgowan): Check for error cases
}

func TestManifestTags(t *testing.T) {
	repo := "test.example.com/repo/tags/list"
	tagsList := []byte(strings.TrimSpace(`
{
	"name": "test.example.com/repo/tags/list",
	"tags": [
		"tag1",
		"tag2",
		"funtag"
	]
}
	`))
	var m testutil.RequestResponseMap
	addPing(&m)
	m = append(m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "GET",
			Route:  "/v2/" + repo + "/tags/list",
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Body:       tagsList,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(len(tagsList))},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})

	e, c := testServer(m)
	defer c()

	r, err := NewRepository(context.Background(), repo, e, nil)
	if err != nil {
		t.Fatal(err)
	}

	ms := r.Manifests()
	tags, err := ms.Tags()
	if err != nil {
		t.Fatal(err)
	}

	if len(tags) != 3 {
		t.Fatalf("Wrong number of tags returned: %d, expected 3", len(tags))
	}
	// TODO(dmcgowan): Check array

	// TODO(dmcgowan): Check for error cases
}
