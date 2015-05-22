package graph

import (
	"fmt"
	"io"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/registry"
)

type ImagePushConfig struct {
	MetaHeaders map[string][]string
	AuthConfig  *cliconfig.AuthConfig
	Tag         string
	OutStream   io.Writer
}

type Pusher interface {
	Push(localName, repoName string, endpoint registry.APIEndpoint, imagePushConfig *ImagePushConfig, sf *streamformatter.StreamFormatter) (fallback bool, err error)
}

func (s *TagStore) NewPusher(endpoint APIEndpoint) Pusher {
	switch endpoint.Version {
	case registry.APIVersion2:
		return &v2Pusher{s}
	case registry.APIVersion1:
		return &v1Pusher{s}
	}
	return fmt.Errorf("unknown version %d for registry %s", endpoint.Version, endpoint.URL)
}

// FIXME: Allow to interrupt current push when new push of same image is done.
func (s *TagStore) Push(localName string, imagePushConfig *ImagePushConfig) error {
	var sf = streamformatter.NewJSONStreamFormatter()

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := s.registryService.ResolveRepository(localName)
	if err != nil {
		return err
	}

	image := CanonicalizeName(localName)

	endpoints, err := s.registryService.LookupEndpoints(image)
	if err != nil {
		return err
	}

	reposLen := 1
	if imagePushConfig.Tag == "" {
		reposLen = len(s.Repositories[repoInfo.LocalName])
	}

	imagePushConfig.OutStream.Write(sf.FormatStatus("", "The push refers to a repository [%s] (len: %d)", repoInfo.CanonicalName, reposLen))

	// If it fails, try to get the repository
	localRepo, exists := s.Repositories[repoInfo.LocalName]
	if !exists {
		return fmt.Errorf("Repository does not exist: %s", repoInfo.LocalName)
	}

	var lastErr error
	for _, endpoint := range endpoints {
		name := image
		logrus.Debugf("Trying to push %s to %s %s", name, endpoint.URL, endpoint.Version)
		// TODO(tiborvass): isn't the following redundant with ResolveRepository?
		if endpoint.TrimHostname {
			if i := strings.IndexRune(name, '/'); i > 0 {
				name = name[i+1:]
			}
		}

		pusher, err := s.NewPusher(endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		if fallback, err := pusher.Push(localName, name, tag, endpoint, imagePushConfig, sf); err != nil {
			if fallback {
				lastErr = err
				continue
			}
			logrus.Debugf("Not continuing with error: %v", err)
			return err

		}
		/*
			switch endpoint.Version {
			case registry.APIVersion2:
			case registry.APIVersion1:
			default:
				lastErr = fmt.Errorf("unknown version %d for registry %s", endpoint.Version, endpoint.URL)
				continue
			}
		*/

		s.eventsService.Log("push", repoInfo.LocalName, "")
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no endpoints found for %s", repoInfo.CanonicalName)
	}
	return lastErr
}
