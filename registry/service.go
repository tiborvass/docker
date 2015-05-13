package registry

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
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
	endpoint, err := NewEndpoint(index, nil)
	if err != nil {
		return "", err
	}
	authConfig.ServerAddress = endpoint.String()
	return Login(authConfig, endpoint)
}

// Search queries the public registry for images matching the specified
// search terms, and returns the results.
func (s *Service) Search(term string, authConfig *cliconfig.AuthConfig, headers map[string][]string) (*SearchResults, error) {
	repoInfo, err := s.ResolveRepository(term)
	if err != nil {
		return nil, err
	}

	// *TODO: Search multiple indexes.
	endpoint, err := repoInfo.GetEndpoint(http.Header(headers))
	if err != nil {
		return nil, err
	}
	r, err := NewSession(endpoint.client, authConfig, endpoint)
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

type APIEndpoint struct {
	URL          string
	Version      APIVersion
	TrimHostname bool
	TLSConfig    *tls.Config
	PullFallback func(error) bool
	PushFallback func(error) bool
	Mirrors      []string
}

func (endpoint APIEndpoint) ping() error {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			Proxy:             http.ProxyFromEnvironment,
			TLSClientConfig:   endpoint.TLSConfig,
		},
		CheckRedirect: AddRequiredHeadersToRedirectedRequests,
		Timeout:       5 * time.Second,
	}
	var path string
	switch endpoint.Version {
	case APIVersion1:
		path = "/v1/_ping"
	case APIVersion2:
		path = "/v2/"
	default:
		return fmt.Errorf("unknown registry API version %d", endpoint.Version)
	}
	resp, err := client.Get(endpoint.URL + path)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return nil
}

func ifTLSError(err error) bool {
	return strings.Contains(err.Error(), "tls: oversized record received with length")
}

func alwaysFallback(error) bool {
	return true
}

func neverFallback(err error) bool {
	return false
}

func (s *Service) LookupEndpoints(repoName string) ([]APIEndpoint, error) {
	if strings.HasPrefix(repoName, "docker.io/") {
		return []APIEndpoint{
			{
				URL:          "https://registry-1.docker.io",
				Version:      APIVersion2,
				TrimHostname: true,
				PullFallback: alwaysFallback,
				PushFallback: neverFallback,
			},
			{
				URL:          "https://index.docker.io",
				Version:      APIVersion1,
				TrimHostname: true,
				PullFallback: alwaysFallback,
				PushFallback: neverFallback,
			},
		}, nil
	}

	slashIndex := strings.IndexRune(repoName, '/')
	if slashIndex <= 0 {
		return nil, fmt.Errorf("invalid repo name: missing '/':  %s", repoName)
	}
	hostname := repoName[:slashIndex]
	isSecure := s.Config.isSecureIndex(hostname)

	tlsConfig := &tls.Config{
		InsecureSkipVerify:       !isSecure,
		MinVersion:               tls.VersionTLS10,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
	if isSecure {
		hasFile := func(files []os.FileInfo, name string) bool {
			for _, f := range files {
				if f.Name() == name {
					return true
				}
			}
			return false
		}

		hostDir := path.Join("/etc/docker/certs.d", hostname)
		logrus.Debugf("hostDir: %s", hostDir)
		fs, err := ioutil.ReadDir(hostDir)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		for _, f := range fs {
			if strings.HasSuffix(f.Name(), ".crt") {
				if tlsConfig.RootCAs == nil {
					// TODO(dmcgowan): Copy system pool
					tlsConfig.RootCAs = x509.NewCertPool()
				}
				logrus.Debugf("crt: %s", hostDir+"/"+f.Name())
				data, err := ioutil.ReadFile(path.Join(hostDir, f.Name()))
				if err != nil {
					return nil, err
				}
				tlsConfig.RootCAs.AppendCertsFromPEM(data)
			}
			if strings.HasSuffix(f.Name(), ".cert") {
				certName := f.Name()
				keyName := certName[:len(certName)-5] + ".key"
				logrus.Debugf("cert: %s", hostDir+"/"+f.Name())
				if !hasFile(fs, keyName) {
					return nil, fmt.Errorf("Missing key %s for certificate %s", keyName, certName)
				}
				cert, err := tls.LoadX509KeyPair(path.Join(hostDir, certName), path.Join(hostDir, keyName))
				if err != nil {
					return nil, err
				}
				tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
			}
			if strings.HasSuffix(f.Name(), ".key") {
				keyName := f.Name()
				certName := keyName[:len(keyName)-4] + ".cert"
				logrus.Debugf("key: %s", hostDir+"/"+f.Name())
				if !hasFile(fs, certName) {
					return nil, fmt.Errorf("Missing certificate %s for key %s", certName, keyName)
				}
			}
		}
	}

	// TODO Get mirrors flag
	endpoints := []APIEndpoint{
		{
			URL:          "https://" + hostname,
			Version:      APIVersion2,
			TrimHostname: true,
			PullFallback: alwaysFallback,
			PushFallback: neverFallback,
			TLSConfig:    tlsConfig,
		},
		{
			URL:          "https://" + hostname,
			Version:      APIVersion1,
			TrimHostname: true,
			PullFallback: alwaysFallback,
			PushFallback: neverFallback,
			TLSConfig:    tlsConfig,
		},
	}

	// TODO(tiborvass): parallelize ping checks
	n := len(endpoints)
	// for each HTTPS endpoint, ping them to make sure they are usable.
	for i := 0; i < n; i++ {
		if endpoints[i].ping() == nil {
			// HTTPS endpoint worked, keep it.
			continue
		}
		// copy bad endpoint
		httpEndpoint := endpoints[i]
		// remove bad endpoint
		endpoints = append(endpoints[:i], endpoints[i+1:]...)
		n--
		i--
		// because HTTPS failed and registry is marked as insecure,
		// let's try to ping via HTTP
		if !isSecure {
			httpEndpoint.URL = "http://" + hostname
			httpEndpoint.TLSConfig = nil

			// if HTTP succeeded, keep it.
			if httpEndpoint.ping() == nil {
				endpoints = append(endpoints, httpEndpoint)
			}
		}
	}

	return endpoints, nil
}
