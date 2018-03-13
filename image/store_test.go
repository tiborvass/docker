package image // import "github.com/tiborvass/docker/image"

import (
	"runtime"
	"testing"

	"github.com/tiborvass/docker/internal/testutil"
	"github.com/tiborvass/docker/layer"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/opencontainers/go-digest"
)

func TestRestore(t *testing.T) {
	fs, cleanup := defaultFSStoreBackend(t)
	defer cleanup()

	id1, err := fs.Set([]byte(`{"comment": "abc", "rootfs": {"type": "layers"}}`))
	assert.Check(t, err)

	_, err = fs.Set([]byte(`invalid`))
	assert.Check(t, err)

	id2, err := fs.Set([]byte(`{"comment": "def", "rootfs": {"type": "layers", "diff_ids": ["2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"]}}`))
	assert.Check(t, err)

	err = fs.SetMetadata(id2, "parent", []byte(id1))
	assert.Check(t, err)

	mlgrMap := make(map[string]LayerGetReleaser)
	mlgrMap[runtime.GOOS] = &mockLayerGetReleaser{}
	is, err := NewImageStore(fs, mlgrMap)
	assert.Check(t, err)

	assert.Check(t, is.Len(is.Map(), 2))

	img1, err := is.Get(ID(id1))
	assert.Check(t, err)
	assert.Check(t, is.Equal(ID(id1), img1.computedID))
	assert.Check(t, is.Equal(string(id1), img1.computedID.String()))

	img2, err := is.Get(ID(id2))
	assert.Check(t, err)
	assert.Check(t, is.Equal("abc", img1.Comment))
	assert.Check(t, is.Equal("def", img2.Comment))

	_, err = is.GetParent(ID(id1))
	testutil.ErrorContains(t, err, "failed to read metadata")

	p, err := is.GetParent(ID(id2))
	assert.Check(t, err)
	assert.Check(t, is.Equal(ID(id1), p))

	children := is.Children(ID(id1))
	assert.Check(t, is.Len(children, 1))
	assert.Check(t, is.Equal(ID(id2), children[0]))
	assert.Check(t, is.Len(is.Heads(), 1))

	sid1, err := is.Search(string(id1)[:10])
	assert.Check(t, err)
	assert.Check(t, is.Equal(ID(id1), sid1))

	sid1, err = is.Search(digest.Digest(id1).Hex()[:6])
	assert.Check(t, err)
	assert.Check(t, is.Equal(ID(id1), sid1))

	invalidPattern := digest.Digest(id1).Hex()[1:6]
	_, err = is.Search(invalidPattern)
	testutil.ErrorContains(t, err, "No such image")
}

func TestAddDelete(t *testing.T) {
	is, cleanup := defaultImageStore(t)
	defer cleanup()

	id1, err := is.Create([]byte(`{"comment": "abc", "rootfs": {"type": "layers", "diff_ids": ["2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"]}}`))
	assert.Check(t, err)
	assert.Check(t, is.Equal(ID("sha256:8d25a9c45df515f9d0fe8e4a6b1c64dd3b965a84790ddbcc7954bb9bc89eb993"), id1))

	img, err := is.Get(id1)
	assert.Check(t, err)
	assert.Check(t, is.Equal("abc", img.Comment))

	id2, err := is.Create([]byte(`{"comment": "def", "rootfs": {"type": "layers", "diff_ids": ["2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"]}}`))
	assert.Check(t, err)

	err = is.SetParent(id2, id1)
	assert.Check(t, err)

	pid1, err := is.GetParent(id2)
	assert.Check(t, err)
	assert.Check(t, is.Equal(pid1, id1))

	_, err = is.Delete(id1)
	assert.Check(t, err)

	_, err = is.Get(id1)
	testutil.ErrorContains(t, err, "failed to get digest")

	_, err = is.Get(id2)
	assert.Check(t, err)

	_, err = is.GetParent(id2)
	testutil.ErrorContains(t, err, "failed to read metadata")
}

func TestSearchAfterDelete(t *testing.T) {
	is, cleanup := defaultImageStore(t)
	defer cleanup()

	id, err := is.Create([]byte(`{"comment": "abc", "rootfs": {"type": "layers"}}`))
	assert.Check(t, err)

	id1, err := is.Search(string(id)[:15])
	assert.Check(t, err)
	assert.Check(t, is.Equal(id1, id))

	_, err = is.Delete(id)
	assert.Check(t, err)

	_, err = is.Search(string(id)[:15])
	testutil.ErrorContains(t, err, "No such image")
}

func TestParentReset(t *testing.T) {
	is, cleanup := defaultImageStore(t)
	defer cleanup()

	id, err := is.Create([]byte(`{"comment": "abc1", "rootfs": {"type": "layers"}}`))
	assert.Check(t, err)

	id2, err := is.Create([]byte(`{"comment": "abc2", "rootfs": {"type": "layers"}}`))
	assert.Check(t, err)

	id3, err := is.Create([]byte(`{"comment": "abc3", "rootfs": {"type": "layers"}}`))
	assert.Check(t, err)

	assert.Check(t, is.SetParent(id, id2))
	assert.Check(t, is.Len(is.Children(id2), 1))

	assert.Check(t, is.SetParent(id, id3))
	assert.Check(t, is.Len(is.Children(id2), 0))
	assert.Check(t, is.Len(is.Children(id3), 1))
}

func defaultImageStore(t *testing.T) (Store, func()) {
	fsBackend, cleanup := defaultFSStoreBackend(t)

	mlgrMap := make(map[string]LayerGetReleaser)
	mlgrMap[runtime.GOOS] = &mockLayerGetReleaser{}
	store, err := NewImageStore(fsBackend, mlgrMap)
	assert.Check(t, err)

	return store, cleanup
}

func TestGetAndSetLastUpdated(t *testing.T) {
	store, cleanup := defaultImageStore(t)
	defer cleanup()

	id, err := store.Create([]byte(`{"comment": "abc1", "rootfs": {"type": "layers"}}`))
	assert.Check(t, err)

	updated, err := store.GetLastUpdated(id)
	assert.Check(t, err)
	assert.Check(t, is.Equal(updated.IsZero(), true))

	assert.Check(t, store.SetLastUpdated(id))

	updated, err = store.GetLastUpdated(id)
	assert.Check(t, err)
	assert.Check(t, is.Equal(updated.IsZero(), false))
}

type mockLayerGetReleaser struct{}

func (ls *mockLayerGetReleaser) Get(layer.ChainID) (layer.Layer, error) {
	return nil, nil
}

func (ls *mockLayerGetReleaser) Release(layer.Layer) ([]layer.Metadata, error) {
	return nil, nil
}
