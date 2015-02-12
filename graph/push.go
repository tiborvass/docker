package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/transport"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/runconfig"
)

type ImagePushConfig struct {
	MetaHeaders map[string][]string
	AuthConfig  *cliconfig.AuthConfig
	Tag         string
	OutStream   io.Writer
}

func (s *TagStore) pushImageRepository(repo distribution.Repository, localRepo Repository, tags []string, out io.Writer, sf *streamformatter.StreamFormatter) error {
	for _, tag := range tags {
		logrus.Debugf("Pushing repository: %s:%s", repo.Name(), tag)

		layerId, exists := localRepo[tag]
		if !exists {
			return fmt.Errorf("tag does not exist: %s", tag)
		}
		layersSeen := make(map[string]bool)

		layer, err := s.graph.Get(layerId)
		if err != nil {
			return err
		}

		m := &manifest.Manifest{
			Versioned: manifest.Versioned{
				SchemaVersion: 1,
			},
			Name:         repo.Name(),
			Tag:          tag,
			Architecture: layer.Architecture,
			FSLayers:     make([]manifest.FSLayer, 0, 4),
			History:      make([]manifest.History, 0, 4),
		}
		var metadata runconfig.Config
		if layer.Config != nil {
			metadata = *layer.Config
		}

		for ; layer != nil; layer, err = layer.GetParent() {
			if err != nil {
				return err
			}

			if layersSeen[layer.ID] {
				break
			}

			if layer.Config != nil && metadata.Image != layer.ID {
				if err := runconfig.Merge(&metadata, layer.Config); err != nil {
					return err
				}
			}

			jsonData, err := layer.RawJson()
			if err != nil {
				return fmt.Errorf("cannot retrieve the path for %s: %s", layer.ID, err)
			}

			// Push v1 compatibility object for compatibility registry
			// Set jsonData and layer.ID

			checksum, err := layer.GetCheckSum(s.graph.ImageRoot(layer.ID))
			if err != nil {
				return fmt.Errorf("error getting image checksum: %s", err)
			}

			var d digest.Digest
			if checksum != "" {
				parsedDigest, err := digest.ParseDigest(checksum)
				if err != nil {
					logrus.Infof("Could not parse digest(%s) for %s: %s", checksum, layer.ID, err)
				}
				d = parsedDigest
			}
			if len(d) > 0 {
				exists, err := repo.Layers().Exists(d)
				if err != nil {
					out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image push failed", nil))
					return fmt.Errorf("error checking if layer exists: %s", err)
				}
				if !exists {
					d = ""
				}
			}

			if len(d) == 0 {
				var err error
				d, err = s.pushImageBlob(repo.Layers(), layer, sf, out)
				if err != nil {
					out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image push failed", nil))
					return err
				}
				out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image successfully pushed", nil))

				// Cache digest
				if err := layer.SaveCheckSum(s.graph.ImageRoot(layer.ID), d.String()); err != nil {
					return err
				}
			} else {
				out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image already exists", nil))
			}

			m.FSLayers = append(m.FSLayers, manifest.FSLayer{BlobSum: d})
			m.History = append(m.History, manifest.History{V1Compatibility: string(jsonData)})

			layersSeen[layer.ID] = true
		}

		logrus.Infof("Signed manifest for %s:%s using daemon's key: %s", repo.Name(), tag, s.trustKey.KeyID())
		signed, err := manifest.Sign(m, s.trustKey)
		if err != nil {
			return err
		}

		if err := repo.Manifests().Put(signed); err != nil {
			return err
		}

	}
	return nil
}

func (s *TagStore) pushImageBlob(ls distribution.LayerService, img *image.Image, sf *streamformatter.StreamFormatter, out io.Writer) (digest.Digest, error) {
	out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Buffering to Disk", nil))

	arch, err := img.TarLayer()
	if err != nil {
		return "", err
	}

	tf, err := s.graph.newTempFile()
	if err != nil {
		return "", err
	}
	defer func() {
		tf.Close()
		os.Remove(tf.Name())
	}()

	size, dgst, err := bufferToFile(tf, arch)
	if err != nil {
		return "", err
	}

	// Send the layer
	logrus.Debugf("rendered layer for %s of [%d] size", img.ID, size)
	layerUpload, err := ls.Upload()
	if err != nil {
		return "", err
	}

	reader := progressreader.New(progressreader.Config{
		In:        ioutil.NopCloser(tf),
		Out:       out,
		Formatter: sf,
		Size:      int(size),
		NewLines:  false,
		ID:        stringid.TruncateID(img.ID),
		Action:    "Pushing",
	})
	n, err := layerUpload.ReadFrom(reader)
	if err != nil {
		return "", err
	}
	if n != size {
		return "", fmt.Errorf("short upload: only wrote %d of %d", n, size)
	}

	if _, err := layerUpload.Finish(dgst); err != nil {
		return "", err
	}

	return dgst, nil
}

func (s *TagStore) getImageTags(localRepo Repository, askedTag string) ([]string, error) {
	if len(askedTag) > 0 {
		if _, ok := localRepo[askedTag]; !ok {
			return nil, fmt.Errorf("Tag does not exist for %s", askedTag)
		}
		return []string{askedTag}, nil
	}
	var tags []string
	for tag := range localRepo {
		tags = append(tags, tag)
	}
	return tags, nil
}

// FIXME: Allow to interrupt current push when new push of same image is done.
func (s *TagStore) Push(localName string, imagePushConfig *ImagePushConfig) error {
	var (
		sf = streamformatter.NewJSONStreamFormatter()
	)

	// Get endpoints to try
	repoName := CanonicalizeName(localName)

	endpoints, err := s.registryService.LookupEndpoints(repoName)
	if err != nil {
		return err
	}

	var lastErr error
	for _, endpoint := range endpoints {
		logrus.Debugf("Trying push to %s %s", endpoint.URL, endpoint.Version)
		name := repoName
		if endpoint.TrimHostname {
			if i := strings.IndexRune(name, '/'); i > 0 {
				name = name[i+1:]
			}
		}

		shouldContinue, err := s.pushImageToEndpoint(localName, name, endpoint, imagePushConfig, sf)
		if shouldContinue {
			lastErr = err
			continue
		}

		return err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no endpoints found for %s", repoName)
	}
	return lastErr
}

func (s *TagStore) pushImageToEndpoint(localName, repoName string, endpoint registry.APIEndpoint, imagePushConfig *ImagePushConfig, sf *streamformatter.StreamFormatter) (bool, error) {
	repo, err := NewRepositoryClient(repoName, endpoint, imagePushConfig.MetaHeaders, imagePushConfig.AuthConfig)
	if err != nil {
		return false, err
	}

	if _, err := s.poolAdd("push", localName); err != nil {
		return false, err
	}
	defer s.poolRemove("push", localName)

	localRepo, err := s.Get(localName)
	if err != nil {
		return false, err
	}
	// TODO(tiborvass): reuse client from endpoint?
	// Adds Docker-specific headers as well as user-specified headers (metaHeaders)
	tr := transport.NewTransport(
		registry.NewTransport(registry.NoTimeout, endpoint.IsSecure),
		registry.DockerHeaders(imagePushConfig.MetaHeaders)...,
	)
	client := registry.HTTPClient(tr)
	r, err := registry.NewSession(client, imagePushConfig.AuthConfig, endpoint)
	if err != nil {
		return endpoint.PushFallback(err), fmt.Errorf("error getting tags for %s: %s", localName, err)
	}
	if len(tags) == 0 {
		return false, fmt.Errorf("no tags to push for %s", localName)
	}

	if err := s.pushImageRepository(repo, localRepo, tags, imagePushConfig.OutStream, sf); err != nil {
		return endpoint.PushFallback(err), err
	}

	return false, nil
}
