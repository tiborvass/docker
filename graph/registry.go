// TODO: move to registry/

package graph

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/transport"
	_ "github.com/docker/distribution/registry/storage/driver/filesystem"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/registry"
)

func isRegistryName(name string) bool {
	return (strings.Contains(name, ".") ||
		strings.Contains(name, ":") ||
		name == "localhost")
}

// CanonicalizeName converts the local representation of the name in
// the Docker graph to a fully namespaced value which includes
// registry target.
func CanonicalizeName(name string) string {
	nameParts := strings.SplitN(name, "/", 2)
	var registryName, repoName string
	if len(nameParts) == 1 || !isRegistryName(nameParts[0]) {
		// Default to official registry
		registryName = "docker.io"
		if len(nameParts) == 1 {
			repoName = "library/" + name
		} else {
			repoName = name
		}
	} else {
		registryName = nameParts[0]
		repoName = nameParts[1]
	}
	return registryName + "/" + repoName
}

type dumbCredentialStore struct {
	auth *cliconfig.AuthConfig
}

func (dcs dumbCredentialStore) Basic(*url.URL) (string, string) {
	return dcs.auth.Username, dcs.auth.Password
}

// v2 only
func NewV2Repository(repoName string, endpoint registry.APIEndpoint, metaHeaders http.Header, auth *cliconfig.AuthConfig) (distribution.Repository, error) {
	ctx := context.Background()

	tokenScope := transport.TokenScope{
		Resource: "repository",
		Scope:    repoName,
		Actions:  []string{"push", "pull"},
	}

	// TODO(dmcgowan): Call close idle connections when complete, use keep alive
	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     endpoint.TLSConfig,
		// TODO(dmcgowan): Call close idle connections when complete and use keep alive
		DisableKeepAlives: true,
	}

	modifiers := registry.DockerHeaders(metaHeaders)
	authTransport := transport.NewTransport(base, modifiers...)
	tokenHandler := transport.NewTokenHandler(authTransport, dumbCredentialStore{auth: auth}, tokenScope)
	modifiers = append(modifiers, transport.NewAuthorizer(authTransport, tokenHandler))
	tr := transport.NewTransport(base, modifiers...)

	return client.NewRepository(ctx, repoName, endpoint.URL, tr)
}

func digestFromManifest(m *manifest.SignedManifest, localName string) (digest.Digest, error) {
	payload, err := m.Payload()
	if err != nil {
		return "", registry.WrapError("could not retrieve manifest payload", err)
	}
	manifestDigest, err := digest.FromBytes(payload)
	if err != nil {
		logrus.Infof("Could not compute manifest digest for %s:%s : %v", localName, m.Tag, err)
	}
	return manifestDigest, nil
}
