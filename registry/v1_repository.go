package registry

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/cliconfig"
)

type v1Authorizer struct {
	cfg    *cliconfig.AuthConfig
	tokens []string
}

func (auth *v1Authorizer) Authorize(req *http.Request) {
	if auth.tokens == nil {
		req.Header.Set("X-Docker-Token", "true")
		return
	}
	if len(req.Header.Get("Authorization")) == 0 {
		req.Header.Set("Authorization", "Token "+strings.Join(auth.tokens, ","))
	}
}

// NewV1Repository creates a new v1 Repository with a v2 API, for the given repository name and endpoint
func NewV1Repository(name string, endpoint APIEndpoint, metaHeaders http.Header, authConfig *cliconfig.AuthConfig) (distribution.Repository, error) {
	if err := v2.ValidateRespositoryName(name); err != nil {
		return nil, err
	}

	secure := endpoint.TLSConfig != nil && !endpoint.TLSConfig.InsecureSkipVerify

	ep, err := newEndpoint(endpoint.URL, secure, metaHeaders)
	if err != nil {
		return nil, err
	}

	// TODO(tiborvass): reuse client from endpoint?
	tr := NewTransport(ReceiveTimeout, ep.IsSecure)
	client := HTTPClient(transport.NewTransport(tr, DockerHeaders(metaHeaders)...))
	session, err := NewSession(client, authConfig, ep)
	if err != nil {
		return nil, err
	}

	return &repository{
		name:     name,
		endpoint: endpoint,
		session:  session,
		sizes:    make(map[string]int64),
		jsonData: make(map[string][]byte),
		imgIndex: make(map[string][]*ImgData),
	}, nil
}

type repository struct {
	name            string
	endpoint        APIEndpoint
	currentEndpoint string
	context         context.Context
	session         *Session
	repoData        *RepositoryData

	// hack
	sizes    map[string]int64
	jsonData map[string][]byte
	imgIndex map[string][]*ImgData
}

type errs map[string]error

func (errs errs) Error() string {
	return fmt.Sprintf("loopEndpoints erros: %v", errs)
}

// loopEndpoints executes a callback function for each endpoint and returns as soon as one succeeds.
// v1 mirrors are tried first, then the list of endpoints returned by the v1 index.
// If none succeed, all errors are returned in an `errs` map.
func (r *repository) loopEndpoints(withV1Mirrors bool, fn func(endpoint string) error) (bool, errs) {
	errs := make(errs)
	defer func() {
		if len(errs) > 0 {
			logrus.Debug(errs)
		}
	}()
	if withV1Mirrors {
		for _, mirror := range r.endpoint.Mirrors {
			err := fn(mirror)
			if err == nil {
				return true, errs
			}
			errs[mirror] = err
		}
	}
	for _, endpoint := range r.repoData.Endpoints {
		err := fn(endpoint)
		if err == nil {
			return true, errs
		}
		errs[endpoint] = err
	}
	return false, errs
}

func (r *repository) Name() string {
	return r.name
}

func (r *repository) Manifests() distribution.ManifestService {
	return &manifests{repository: r}
}

func (r *repository) Blobs(ctx context.Context) distribution.BlobStore {
	return &blobs{repository: r}
}

func (r *repository) Signatures() distribution.SignatureService {
	return nil
}

type PushPreparer interface {
	PreparePush(imgID string, jsonData []byte)
}

func (r *repository) PreparePush(imgID string, jsonData []byte) {
	r.jsonData[imgID] = jsonData
}

type PushInitializer interface {
	InitPush(localName string, imgList []string, tagsByImage map[string][]string) error
}

func (r *repository) InitPush(localName string, imgList []string, tagsByImage map[string][]string) (err error) {
	r.imgIndex[localName] = createImageIndex(imgList, tagsByImage)
	// Register all the images in a repository with the registry
	// If an image is not in this list it will not be associated with the repository
	r.repoData, err = r.session.PushImageJSONIndex(r.name, r.imgIndex[localName], false, nil)
	return
}

type PushFinalizer interface {
	FinalizePush(localName string) error
}

func (r *repository) FinalizePush(localName string) error {
	imgIndex, ok := r.imgIndex[localName]
	if !ok {
		return fmt.Errorf("could not find imgIndex for %s. Make sure you called InitPush", localName)
	}
	_, err := r.session.PushImageJSONIndex(r.name, imgIndex, true, r.repoData.Endpoints)
	return err
}

type DigestConverter interface {
	ConvertToDigest(id string) digest.Digest
	ConvertFromDigest(digest.Digest) string
}

func (r *repository) ConvertToDigest(id string) digest.Digest {
	return imgIDToDigest(id)
}

func (r *repository) ConvertFromDigest(d digest.Digest) string {
	return digestToImgID(d)
}

func createImageIndex(images []string, tags map[string][]string) []*ImgData {
	var imageIndex []*ImgData
	for _, id := range images {
		if tags, hasTags := tags[id]; hasTags {
			// If an image has tags you must add an entry in the image index
			// for each tag
			for _, tag := range tags {
				imageIndex = append(imageIndex, &ImgData{
					ID:  id,
					Tag: tag,
				})
			}
			continue
		}
		// If the image does not have a tag it still needs to be sent to the
		// registry with an empty tag so that it is accociated with the repository
		imageIndex = append(imageIndex, &ImgData{
			ID:  id,
			Tag: "",
		})
	}
	return imageIndex
}
