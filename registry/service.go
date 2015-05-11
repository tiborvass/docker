package registry

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/docker/docker/cliconfig"
)

type Service struct {
	Config *ServiceConfig
}

// NewService returns a new instance of Service ready to be
// installed no an engine.
func NewService(options *Options) *Service {
	return &Service{
		Config: NewServiceConfig(options),
	}
}

// Auth contacts the public registry with the provided credentials,
// and returns OK if authentication was sucessful.
// It can be used to verify the validity of a client's credentials.
func (s *Service) Auth(authConfig *cliconfig.AuthConfig) (string, error) {
	addr := authConfig.ServerAddress
	if addr == "" {
		// Use the official registry address if not specified.
		addr = IndexServerAddress()
	}
	index, err := s.ResolveIndex(addr)
	if err != nil {
		return "", err
	}
	endpoint, err := NewEndpoint(index)
	if err != nil {
		return "", err
	}
	authConfig.ServerAddress = endpoint.String()
	return Login(authConfig, endpoint, HTTPRequestFactory(nil))
}

// Search queries the public registry for images matching the specified
// search terms, and returns the results.
func (s *Service) Search(term string, authConfig *cliconfig.AuthConfig, headers map[string][]string) (*SearchResults, error) {
	repoInfo, err := s.ResolveRepository(term)
	if err != nil {
		return nil, err
	}
	// *TODO: Search multiple indexes.
	endpoint, err := repoInfo.GetEndpoint()
	if err != nil {
		return nil, err
	}
	r, err := NewSession(authConfig, HTTPRequestFactory(headers), endpoint, true)
	if err != nil {
		return nil, err
	}
	return r.SearchRepositories(repoInfo.GetSearchTerm())
}

// ResolveRepository splits a repository name into its components
// and configuration of the associated registry.
func (s *Service) ResolveRepository(name string) (*RepositoryInfo, error) {
	return s.Config.NewRepositoryInfo(name)
}

// ResolveIndex takes indexName and returns index info
func (s *Service) ResolveIndex(name string) (*IndexInfo, error) {
	return s.Config.NewIndexInfo(name)
}

func hostToEndpoint(host, version string) (u *url.URL, trimHost bool) {
	if host == "docker.io" {
		host = "index.docker.io"
		// makes it so that docker.io/library/ubuntu becomes library/ubuntu for the image name in the URL
		trimHost = true
	}
	return &url.URL{
		Host:   host,
		Scheme: "https",
	}, trimHost
}

// NewRepository
func (s *Service) NewRepository(canonicalRepoName, action string, metaHeaders map[string][]string, authConfig *cliconfig.AuthConfig) (Repository, error) {
	// set up [][]endpoint based on insecure registries and mirrors

	// extract host, first part of canonical repository name
	i := strings.IndexByte(canonicalRepoName, '/')
	if i < 0 {
		return nil, fmt.Errorf("Expecting fully qualified repository name, got %s", canonicalRepoName)
	}
	host := canonicalRepoName[:i]

	indexCfg := s.Config.IndexConfigs[host]

	repo := make(fallbackRepository, 0, 2)

	common := &commonRepository{
		action:      action,
		metaHeaders: metaHeaders,
		authConfig:  authConfig,
	}
	//TODO: make v2 variable
	endpoint, trimHost := hostToEndpoint(host, "v2")
	if trimHost {
		common.name = canonicalRepoName[i+1:]
	}

	// Only add v2 endpoints if there are no v1 mirrors
	// TODO: update when we support v2 mirrors
	if len(indexCfg.Mirrors) > 0 {
		return nil, errors.New("v1 not implemented")
	}

	if v2repo, err := newV2Repository(common, endpoint); err == nil && len(indexCfg.Mirrors) == 0 {
		repo = append(repo, v2repo)
	}

	//s.Config.InsecureRegistryCIDRs
	return repo, nil
}
