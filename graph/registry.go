package graph

import (
	"net/url"

	_ "github.com/docker/distribution/registry/storage/driver/filesystem"
	"github.com/docker/docker/cliconfig"
)

type dumbCredentialStore struct {
	auth *cliconfig.AuthConfig
}

func (dcs dumbCredentialStore) Basic(*url.URL) (string, string) {
	return dcs.auth.Username, dcs.auth.Password
}
