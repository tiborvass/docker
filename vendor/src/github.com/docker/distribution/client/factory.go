package client

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/context"

	"github.com/docker/distribution"
	"github.com/docker/distribution/namespace"
	rclient "github.com/docker/distribution/registry/client"
)

// RepositoryClientConfig is used to create new clients from endpoints
type RepositoryClientConfig struct {
	TrimHostname bool
	AllowMirrors bool
	Header       http.Header
	RepoScope    string

	Credentials rclient.CredentialStore
	Endpoints   EndpointProvider
}

type scope string

// Contains returns true if the name matches the scope.
func (s scope) Contains(name string) bool {
	// Check for an exact match, with a cleaned path component
	if path.Clean(string(s)) == path.Clean(name) {
		return true
	}

	// A simple prefix match is enough.
	if strings.HasPrefix(name, string(s)) {
		return true
	}

	return false
}

func (s scope) String() string {
	return string(s)
}

// EndpointProvider provides URLs for a given name with the option
// to accept mirrors.
type EndpointProvider func(name string, allowMirrors bool) ([]*url.URL, error)

// StaticEndpointProvider returns an EndpointProvider which always
// returns the same URL.
func StaticEndpointProvider(u *url.URL) EndpointProvider {
	return func(string, bool) ([]*url.URL, error) {
		return []*url.URL{u}, nil
	}
}

// NamespaceEndpointProvider returns an EndpointProvider which
// resolves the name and returns URLs based on that resolution.
func NamespaceEndpointProvider(resolver namespace.Resolver) EndpointProvider {
	return func(name string, allowMirrors bool) ([]*url.URL, error) {
		resolved, err := resolver.Resolve(name)
		if err != nil {
			return nil, err
		}
		endpoints, err := namespace.GetRemoteEndpoints(resolved)
		if err != nil {
			return nil, err
		}
		if len(endpoints) == 0 {
			return nil, errors.New("no endpoints found")
		}
		// TODO(dmcgowan): return prioritized list of endpoints
		// TODO(dmcgowan): only return endpoints with proper action

		return []*url.URL{endpoints[0].BaseURL}, nil
	}
}

// Scope returns the scope for which the configuration is valid
func (f *RepositoryClientConfig) Scope() distribution.Scope {
	if f.RepoScope == "" {
		return distribution.GlobalScope
	}
	return scope(f.RepoScope)
}

// Repository creates a new repository from the configuration
func (f *RepositoryClientConfig) Repository(ctx context.Context, name string) (distribution.Repository, error) {
	if !f.Scope().Contains(name) {
		return nil, errors.New("name out of scope for configuration")
	}
	if f.Endpoints == nil {
		return nil, errors.New("no endpoint provider")
	}
	endpoints, err := f.Endpoints(name, f.AllowMirrors)
	if err != nil {
		return nil, err
	}
	if len(endpoints) == 0 {
		return nil, errors.New("no endpoints")
	}

	if f.TrimHostname {
		i := strings.IndexRune(name, '/')
		if i > -1 && i < len(name)-1 {
			// TODO(dmcgowan): Check if first element is actually hostname
			name = name[i+1:]
		}

	}

	// Currently only single endpoint repository used
	endpoint := &rclient.RepositoryEndpoint{
		Header:      f.Header,
		Credentials: f.Credentials,
		Endpoint:    endpoints[0].String(),
	}

	// TODO(dmcgowan): Support multiple endpoints

	return rclient.NewRepositoryClient(context.Background(), name, endpoint)
}
