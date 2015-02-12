package graph

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/distribution"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/cache"
	"github.com/docker/distribution/registry/storage/driver/factory"
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

func NewRepositoryClient(repoName string, endpoint registry.APIEndpoint, metaHeaders http.Header, auth *cliconfig.AuthConfig) (distribution.Repository, error) {
	ctx := context.Background()

	if localDirectory := os.Getenv("DOCKER_LOCAL_REGISTRY"); localDirectory != "" {
		parameters := map[string]interface{}{
			"rootdirectory": localDirectory,
		}
		driver, err := factory.Create("filesystem", parameters)
		if err != nil {
			return nil, err
		}
		namespace := storage.NewRegistryWithDriver(ctx, driver, cache.NewInMemoryLayerInfoCache())
		return namespace.Repository(ctx, repoName)
	}

	// Call close idle connections when complete
	// TODO(dmcgowan): Setup tls
	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     endpoint.TLSConfig,
	}

	headers := http.Header{}
	for k, v := range headers {
		headers[k] = v
	}
	headers.Add("User-Agent", "docker/1.7.0-dev")

	tokenScope := client.TokenScope{
		Resource: "repository",
		Scope:    repoName,
		Actions:  []string{"push", "pull"},
	}

	authorizer := client.NewTokenAuthorizer(dumbCredentialStore{auth: auth}, base, headers, tokenScope)
	clientConfig := &client.RepositoryConfig{
		Header:     headers,
		AuthSource: authorizer,
	}

	return client.NewRepository(ctx, repoName, endpoint.URL, clientConfig)
}
