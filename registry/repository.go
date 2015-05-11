package registry

import (
	"io"

	"github.com/docker/distribution/digest"
	"github.com/docker/docker/cliconfig"
)

// Repository abstracts away v1 and v2 registry clients
type Repository interface {
	Name() string
	Tags() ([]string, error)
	Layers(tag string) ([]Layer, error)
}

// Layer abstracts away v1 and v2 registry clients
type Layer interface {
	Digest() digest.Digest
	V1Json() (io.ReadCloser, error)
	Fetch() (blob io.ReadCloser, size int64, verify func() bool, err error)
}

type commonRepository struct {
	name        string
	action      string
	metaHeaders map[string][]string
	authConfig  *cliconfig.AuthConfig
}

func (r *commonRepository) Name() string {
	return r.name
}

type fallbackRepository []Repository

func (fr fallbackRepository) Name() string {
	for _, r := range fr {
		if r != nil {
			return r.Name()
		}
	}
	return ""
}

func (fr fallbackRepository) Tags() (tags []string, err error) {
	if err := fr.ensureEndpointsKnown(); err != nil {
		return nil, err
	}
	for _, r := range fr {
		tags, err = r.Tags()
		if err == nil {
			return tags, nil
		}
	}
	return tags, err
}

func (fr fallbackRepository) Layers(tag string) (layers []Layer, err error) {
	if err := fr.ensureEndpointsKnown(); err != nil {
		return nil, err
	}
	for _, r := range fr {
		layers, err = r.Layers(tag)
		if err == nil {
			return layers, nil
		}
	}
	return layers, err
}

func (fr fallbackRepository) ensureEndpointsKnown() error {
	// for loop over [][]endpoint and find the first one that works
	// and cache that []endpoint list for later

	// v1 would save endpoints received via headers instead of its own endpoint
	return nil
}
