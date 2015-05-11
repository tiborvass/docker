package registry

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/url"

	"github.com/docker/distribution/digest"
)

type v1Repository struct {
	*commonRepository
	*Session
	repoData *RepositoryData
	mapTags  map[string]string
}

func newV1Repository(common *commonRepository, endpoint *url.URL) (*v1Repository, error) {
	secure := true
	ep, err := newEndpoint(endpoint.String(), secure)
	if err != nil {
		return nil, err
	}
	session, err := NewSession(common.authConfig, HTTPRequestFactory(common.metaHeaders), ep, true)
	if err != nil {
		return nil, err
	}
	return &v1Repository{commonRepository: common, Session: session}, nil
}

func (r *v1Repository) ensureRepoData() error {
	if r.repoData != nil {
		return nil
	}
	repoData, err := r.GetRepositoryData(r.name)
	if err != nil {
		return err
	}
	r.repoData = repoData
	return nil
}

func (r *v1Repository) Tags() (tags []string, err error) {
	if err := r.ensureRepoData(); err != nil {
		return nil, err
	}
	mapTags, err := r.GetRemoteTags(r.repoData.Endpoints, r.name, r.repoData.Tokens)
	if err != nil {
		return nil, err
	}
	r.mapTags = mapTags
	for tag := range mapTags {
		tags = append(tags, tag)
	}
	return tags, nil
}

func (r *v1Repository) Layers(tag string) (layers []Layer, err error) {
	if err := r.ensureRepoData(); err != nil {
		return nil, err
	}
	var h []string
	for _, ep := range r.repoData.Endpoints {
		h, err = r.GetRemoteHistory(r.mapTags[tag], ep, r.repoData.Tokens)
		if err == nil {
			layers = make([]Layer, len(h))
			for i := range layers {
				layers[i] = v1layer{v1Repository: r, id: h[i]}
			}
			return layers, nil
		}
	}
	return layers, err
}

type v1layer struct {
	*v1Repository
	id   string
	size int64
}

// Digest is not supported for v1
func (l v1layer) Digest() digest.Digest {
	return ""
}

func (l v1layer) V1Json() (io.ReadCloser, error) {
	var (
		json []byte
		size int
		err  error
	)
	for _, ep := range l.repoData.Endpoints {
		json, size, err = l.GetRemoteImageJSON(l.id, ep, l.repoData.Tokens)
		if err == nil {
			l.size = int64(size)
			return ioutil.NopCloser(bytes.NewReader(json)), nil
		}
	}
	return nil, err
}

func (l v1layer) Fetch() (blob io.ReadCloser, size int64, verify func() bool, err error) {
	for _, ep := range l.repoData.Endpoints {
		blob, err = l.GetRemoteImageLayer(l.id, ep, l.repoData.Tokens, l.size)
		if err == nil {
			return blob, l.size, nil, nil
		}
	}
	return nil, 0, nil, err
}
