package graph

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/context"

	"github.com/docker/distribution"
	"github.com/docker/distribution/client"
	"github.com/docker/distribution/namespace"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/cache"
	"github.com/docker/distribution/registry/storage/driver/factory"
	_ "github.com/docker/distribution/registry/storage/driver/filesystem"
	"github.com/docker/docker/cliconfig"
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

func NewRepositoryClient(repoName string, metaHeaders http.Header, auth *cliconfig.AuthConfig) (distribution.Repository, error) {
	if localDirectory := os.Getenv("DOCKER_LOCAL_REGISTRY"); localDirectory != "" {
		parameters := map[string]interface{}{
			"rootdirectory": localDirectory,
		}
		driver, err := factory.Create("filesystem", parameters)
		if err != nil {
			return nil, err
		}
		namespace := storage.NewRegistryWithDriver(driver, cache.NewInMemoryLayerInfoCache())
		return namespace.Repository(context.Background(), repoName)
	}

	if nsFile := os.Getenv("DOCKER_NAMESPACE_CFG"); nsFile != "" {
		headers := http.Header{}
		for k, v := range headers {
			headers[k] = v
		}
		headers.Add("User-Agent", "docker/1.7.0-dev")

		resolver, err := namespace.NewDefaultFileResolver(nsFile)
		if err != nil {
			return nil, err
		}

		// TODO(dmcgowan): Pass in authorization information
		rc := &client.RepositoryClientConfig{
			TrimHostname: true,
			AllowMirrors: true,
			Header:       headers,
			RepoScope:    repoName,
			Credentials:  dumbCredentialStore{auth: auth},
			Endpoints:    client.NamespaceEndpointProvider(resolver),
		}

		return rc.Repository(context.Background(), repoName)
	}

	return nil, errors.New("No supported registry")
}
