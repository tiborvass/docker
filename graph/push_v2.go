package graph

import (
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/runconfig"
)

type v2Pusher struct{ *TagStore }

func (p *v2Pusher) Push(localName, repoName string, endpoint registry.APIEndpoint, imagePushConfig *ImagePushConfig, sf *streamformatter.StreamFormatter) (fallback bool, err error) {
	if err := p.pushV2Repository(localName, name, endpoint, imagePushConfig, sf); err != nil {
		if rErr, ok := err.(*registry.ErrRegistry); ok && rErr.Fallback {
			return true, err
		}
		logrus.Debugf("Not continuing with error: %v", err)
		return false, err
	}
	return false, nil
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

func (s *TagStore) pushV2Repository(localName, repoName string, endpoint registry.APIEndpoint, imagePushConfig *ImagePushConfig, sf *streamformatter.StreamFormatter) error {
	repo, err := NewV2Repository(repoName, endpoint, imagePushConfig.MetaHeaders, imagePushConfig.AuthConfig)
	if err != nil {
		return registry.WrapError("error creating client", err)
	}
	if _, err := s.poolAdd("push", localName); err != nil {
		return err
	}
	defer s.poolRemove("push", localName)

	localRepo, err := s.Get(localName)
	if err != nil {
		return err
	}

	tags, err := s.getImageTags(localRepo, imagePushConfig.Tag)
	if err != nil {
		return fmt.Errorf("error getting tags for %s: %s", localName, err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("no tags to push for %s", localName)
	}

	for _, tag := range tags {
		if err := s.pushV2Tag(repo, localRepo, tag, imagePushConfig.OutStream, sf); err != nil {
			return registry.WrapError("error pushing tag", err)
		}
	}

	return nil
}

func (s *TagStore) pushV2Tag(repo distribution.Repository, localRepo Repository, tag string, out io.Writer, sf *streamformatter.StreamFormatter) error {
	logrus.Debugf("Pushing repository: %s:%s", repo.Name(), tag)

	layerId, exists := localRepo[tag]
	if !exists {
		return fmt.Errorf("tag does not exist: %s", tag)
	}

	// TODO(tiborvass): @dmcgowan isn't this supposed to be outside the tags loop?
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
	if layer != nil && layer.Config != nil {
		metadata = *layer.Config
	}

	for ; layer != nil; layer, err = layer.GetParent() {
		if err != nil {
			return err
		}

		if layersSeen[layer.ID] {
			break
		}

		logrus.Debugf("Pushing layer: %s", layer.ID)

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

		var d digest.Digest
		checksum, err := layer.GetCheckSum(s.graph.ImageRoot(layer.ID))
		if err != nil {
			return fmt.Errorf("error getting image checksum: %s", err)
		}
		if checksum != "" {
			parsedDigest, err := digest.ParseDigest(checksum)
			if err != nil {
				logrus.Infof("Could not parse digest(%s) for %s: %s", checksum, layer.ID, err)
			}
			d = parsedDigest
		}

		if len(d) > 0 {
			if _, err := repo.Blobs(nil).Stat(nil, d); err == nil {
				out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image already exists", nil))
			} else {
				switch err {
				case distribution.ErrBlobUnknown:
					d = ""
				default:
					out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image push failed", nil))
					return registry.WrapError("error checking if layer exists", err)
				}
			}
		}

		// if digest was empty or not saved, or if blob does not exist on the remote repository
		if len(d) == 0 {
			d, err = s.pushV2Image(repo.Blobs(nil), layer, sf, out)
			if err != nil {
				out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image push failed", nil))
				return err
			}

			out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image successfully pushed", nil))

			// Cache digest
			if err := layer.SaveCheckSum(s.graph.ImageRoot(layer.ID), d.String()); err != nil {
				return err
			}
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

	manifestDigest, err := digestFromManifest(signed, repo.Name())
	if err != nil {
		return err
	}
	if manifestDigest != "" {
		out.Write(sf.FormatStatus("", "Digest: %s", manifestDigest))
	}

	return repo.Manifests().Put(signed)
}

func (s *TagStore) pushV2Image(bs distribution.BlobService, img *image.Image, sf *streamformatter.StreamFormatter, out io.Writer) (digest.Digest, error) {
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
	layerUpload, err := bs.Create(nil)
	if err != nil {
		return "", err
	}
	// TODO(tiborvass): @dmcgowan need review
	defer layerUpload.Close()

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

	desc := distribution.Descriptor{Digest: dgst}
	if _, err := layerUpload.Commit(nil, desc); err != nil {
		return "", err
	}

	return dgst, nil
}
