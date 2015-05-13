package graph

import (
	"errors"
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
	"github.com/docker/docker/registry"
	"github.com/docker/docker/runconfig"
	"github.com/docker/docker/utils"
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
			if repo, ok := repo.(registry.PushPreparer); ok {
				repo.PreparePush(layer.ID, jsonData)
			}

			// Push v1 compatibility object for compatibility registry
			// Set jsonData and layer.ID

			var (
				d              digest.Digest
				supportsDigest = true
			)

			// The v1 repository client implements DigestConverter to reuse the digest field to store the layer ID.
			// TODO(tiborvass): remove first block once v1 is retired
			if repo, ok := repo.(registry.DigestConverter); ok {
				supportsDigest = false
				// "random:<layer.ID>"
				// This digest should not make it neither on disk nor on the network.
				d = repo.ConvertToDigest(layer.ID)
			} else {
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
			}

			// if digest was saved or if pushing to v1 repository
			if len(d) > 0 {
				if _, err := repo.Blobs(nil).Stat(nil, d); err == nil {
					out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image already exists", nil))
				} else {
					switch err {
					case distribution.ErrBlobUnknown:
						d = ""
					default:
						out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image push failed", nil))
						return fmt.Errorf("error checking if layer exists: %s", err)
					}
				}
			}

			// if digest was empty or not saved, or if blob does not exist on the remote repository
			if len(d) == 0 {
				d, err = s.pushImageBlob(repo.Blobs(nil), layer, sf, out)
				if err != nil {
					out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image push failed", nil))
					return err
				}

				out.Write(sf.FormatProgress(stringid.TruncateID(layer.ID), "Image successfully pushed", nil))

				// TODO(tiborvass): remove condition once v1 is retired
				if supportsDigest {
					// Cache digest
					if err := layer.SaveCheckSum(s.graph.ImageRoot(layer.ID), d.String()); err != nil {
						return err
					}
				}
			}

			m.FSLayers = append(m.FSLayers, manifest.FSLayer{BlobSum: d})
			m.History = append(m.History, manifest.History{V1Compatibility: string(jsonData)})

			layersSeen[layer.ID] = true
		}

		// TODO(tiborvass): once v1 is retired:
		//	remove manually crafted SignedManifest, only keep manifest.Sign() and remove nil-check condition on repo.Signatures()
		signed := &manifest.SignedManifest{
			Manifest: *m,
			Raw:      []byte(layer.ID),
		}
		if repo.Signatures() != nil {
			logrus.Infof("Signed manifest for %s:%s using daemon's key: %s", repo.Name(), tag, s.trustKey.KeyID())
			var err error
			signed, err = manifest.Sign(m, s.trustKey)
			if err != nil {
				return err
			}
		}

		if err := repo.Manifests().Put(signed); err != nil {
			return err
		}

	}
	return nil
}

func (s *TagStore) pushImageBlob(bs distribution.BlobService, img *image.Image, sf *streamformatter.StreamFormatter, out io.Writer) (digest.Digest, error) {
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
	if n >= 0 && n != size {
		return "", fmt.Errorf("short upload: only wrote %d of %d", n, size)
	}

	if bs, ok := bs.(registry.DigestConverter); ok {
		dgst = bs.ConvertToDigest(img.ID)
	}

	desc := distribution.Descriptor{Digest: dgst}
	if _, err := layerUpload.Commit(nil, desc); err != nil {
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

func (s *TagStore) getImageList(localRepo map[string]string, requestedTag string) ([]string, map[string][]string, error) {
	var (
		imageList   []string
		imagesSeen  = make(map[string]bool)
		tagsByImage = make(map[string][]string)
	)

	for tag, id := range localRepo {
		if requestedTag != "" && requestedTag != tag {
			// Include only the requested tag.
			continue
		}

		if utils.DigestReference(tag) {
			// Ignore digest references.
			continue
		}

		var imageListForThisTag []string

		tagsByImage[id] = append(tagsByImage[id], tag)

		for img, err := s.graph.Get(id); img != nil; img, err = img.GetParent() {
			if err != nil {
				return nil, nil, err
			}

			if imagesSeen[img.ID] {
				// This image is already on the list, we can ignore it and all its parents
				break
			}

			imagesSeen[img.ID] = true
			imageListForThisTag = append(imageListForThisTag, img.ID)
		}

		// reverse the image list for this tag (so the "most"-parent image is first)
		for i, j := 0, len(imageListForThisTag)-1; i < j; i, j = i+1, j-1 {
			imageListForThisTag[i], imageListForThisTag[j] = imageListForThisTag[j], imageListForThisTag[i]
		}

		// append to main image list
		imageList = append(imageList, imageListForThisTag...)
	}
	if len(imageList) == 0 {
		return nil, nil, fmt.Errorf("No images found for the requested repository / tag")
	}
	logrus.Debugf("Image list: %v", imageList)
	logrus.Debugf("Tags by image: %v", tagsByImage)

	return imageList, tagsByImage, nil
}

// FIXME: Allow to interrupt current push when new push of same image is done.
func (s *TagStore) Push(localName string, imagePushConfig *ImagePushConfig) error {
	var (
		sf = streamformatter.NewJSONStreamFormatter()
	)

	// Get endpoints to try
	repoName := CanonicalizeName(localName)

	// TODO(tiborvass): the lookup and loop should be in registry/ not graph/
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

	var tags []string

	// TODO(tiborvass): remove conditional once v1 is retired.
	// delete else block
	if endpoint.Version >= registry.APIVersion2 {
		var err error
		tags, err = s.getImageTags(localRepo, imagePushConfig.Tag)
		if err != nil {
			return endpoint.PushFallback(err), fmt.Errorf("error getting tags for %s: %v", localName, err)
		}
		if len(tags) == 0 {
			return false, fmt.Errorf("no tags to push for %s", localName)
		}
	} else {
		imageList, tagsByImage, err := s.getImageList(localRepo, imagePushConfig.Tag)
		if err != nil {
			return endpoint.PushFallback(err), err
		}
		repo, ok := repo.(registry.PushInitializer)
		if !ok {
			return false, errors.New("unexpected registry client implementation")
		}
		if err := repo.InitPush(localName, imageList, tagsByImage); err != nil {
			return endpoint.PushFallback(err), err
		}
	}

	if err := s.pushImageRepository(repo, localRepo, tags, imagePushConfig.OutStream, sf); err != nil {
		return endpoint.PushFallback(err), err
	}

	if repo, ok := repo.(registry.PushFinalizer); ok {
		if err := repo.FinalizePush(localName); err != nil {
			return endpoint.PushFallback(err), err
		}
	}

	return false, nil
}
