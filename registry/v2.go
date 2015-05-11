package registry

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/utils"
)

type v2Repository struct {
	*commonRepository
	v2        distribution.Repository
	manifests distribution.ManifestService
}

func newV2Repository(common *commonRepository, endpoint *url.URL) (*v2Repository, error) {
	//TODO: just make metaHeaders an http.Header already
	headers := http.Header(common.metaHeaders)
	//TODO: what is this?
	scope := client.TokenScope{}

	cfg := &client.RepositoryConfig{
		Header:       headers,
		AuthSource:   client.NewTokenAuthorizer(common.authConfig, headers, scope),
		AllowMirrors: common.action == "pull",
	}
	repo, err := client.NewRepository(context.Background(), common.name, endpoint, cfg)
	if err != nil {
		return nil, err
	}
	return &v2Repository{
		commonRepository: common,
		v2:               repo,
	}, nil
}

func (r *v2Repository) ensureManifests() {
	if r.manifests == nil {
		r.manifests = r.v2.Manifests()
	}
}

func (r *v2Repository) Tags() (tags []string, err error) {
	r.ensureManifests()
	tags, err = r.manifests.Tags()
	return tags, err
}

type v2layer struct {
	distribution.LayerService
	manifest.FSLayer
	manifest.History
}

func (l v2layer) Digest() digest.Digest {
	return l.BlobSum
}

func (l v2layer) V1Json() (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader(l.V1Compatibility)), nil
}

func (l v2layer) Fetch() (blob io.ReadCloser, size int64, verify func() bool, err error) {
	layer, err := l.LayerService.Fetch(l.BlobSum)
	if err != nil {
		return nil, 0, nil, err
	}
	size, err = layer.Seek(0, os.SEEK_END)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("error seeking to end: %v", err)
	} else if size == 0 {
		return nil, 0, nil, fmt.Errorf("layer did not return a size: %s", l.BlobSum)
	}
	if _, err := layer.Seek(0, 0); err != nil {
		return nil, 0, nil, fmt.Errorf("error seeking to beginning: %s", err)
	}

	verifier, err := digest.NewDigestVerifier(layer.Digest())
	if err != nil {
		return nil, 0, nil, err
	}
	return ioutil.NopCloser(io.TeeReader(layer, verifier)), size, verifier.Verified, nil
}

func (r *v2Repository) Layers(tag string) (layers []Layer, err error) {
	r.ensureManifests()
	manifest, err := r.manifests.GetByTag(tag)
	if err != nil {
		return nil, fmt.Errorf("error getting image manifest: %v", err)
	}
	if manifest == nil {
		return nil, fmt.Errorf("image manifest does not exist for tag: %s", tag)
	}
	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupport image manifest version(%d) for tag: %s", manifest.SchemaVersion, tag)
	}
	if err := checkValidManifest(manifest); err != nil {
		return nil, err
	}
	logrus.Printf("Image manifest for %s has been verified", utils.ImageReference(r.name, tag))
	layers = make([]Layer, len(manifest.FSLayers))
	for i := range layers {
		layers[i] = v2layer{r.v2.Layers(), manifest.FSLayers[i], manifest.History[i]}
	}
	return layers, nil
}

func checkValidManifest(manifest *manifest.SignedManifest) error {
	if len(manifest.FSLayers) != len(manifest.History) {
		return fmt.Errorf("length of history not equal to number of layers")
	}
	if len(manifest.FSLayers) == 0 {
		return fmt.Errorf("no FSLayers in manifest")
	}
	return nil
}
