package graph

import (
	"fmt"
	"io"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

type ImagePullConfig struct {
	MetaHeaders map[string][]string
	AuthConfig  *cliconfig.AuthConfig
	OutStream   io.Writer
}

type Puller interface {
	Pull(image, name, tag string, endpoint registry.APIEndpoint, imagePullConfig *ImagePullConfig, sf *streamformatter.StreamFormatter) (fallback bool, err error)
}

func (s *TagStore) NewPuller(endpoint APIEndpoint) Puller {
	switch endpoint.Version {
	case registry.APIVersion2:
		return &v2Puller{s}
	case registry.APIVersion1:
		return &v1Puller{s}
	}
	return fmt.Errorf("unknown version %d for registry %s", endpoint.Version, endpoint.URL)
}

func (s *TagStore) Pull(image string, tag string, imagePullConfig *ImagePullConfig) error {
	var sf = streamformatter.NewJSONStreamFormatter()

	image = CanonicalizeName(image)

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := s.registryService.ResolveRepository(image)
	if err != nil {
		return err
	}

	// makes sure name is not empty or `scratch`
	if err := validateRepoName(repoInfo.LocalName); err != nil {
		return err
	}

	endpoints, err := s.registryService.LookupEndpoints(image)
	if err != nil {
		return err
	}

	logName := repoInfo.LocalName
	if tag != "" {
		logName = utils.ImageReference(logName, tag)
	}

	var lastErr error
	for _, endpoint := range endpoints {
		name := image
		logrus.Debugf("Trying to pull %s from %s %s", name, endpoint.URL, endpoint.Version)

		if (len(endpoint.Mirrors) == 0 && endpoint.Official) || (endpoint.Version == registry.APIVersion2) {
			if repoInfo.Official {
				s.trustService.UpdateBase()
			}
		}

		// TODO(tiborvass): isn't the following redundant with ResolveRepository?
		if endpoint.TrimHostname {
			if i := strings.IndexRune(name, '/'); i > 0 {
				name = name[i+1:]
			}
		}

		puller, err := s.NewPuller(endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		if fallback, err := puller.Pull(image, name, tag, endpoint, imagePullConfig, sf); err != nil {
			if fallback {
				lastErr = err
				continue
			}
			logrus.Debugf("Not continuing with error: %v", err)
			return err

		}

		s.eventsService.Log("pull", logName, "")
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no endpoints found for %s", image)
	}
	return lastErr
}

func WriteStatus(requestedTag string, out io.Writer, sf *streamformatter.StreamFormatter, layersDownloaded bool) {
	if layersDownloaded {
		out.Write(sf.FormatStatus("", "Status: Downloaded newer image for %s", requestedTag))
	} else {
		out.Write(sf.FormatStatus("", "Status: Image is up to date for %s", requestedTag))
	}
}
