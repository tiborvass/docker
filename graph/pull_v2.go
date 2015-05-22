package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

type v2Puller struct{ *TagStore }

func (p *v2Puller) Pull(image, name, tag string, endpoint registry.APIEndpoint, imagePullConfig *ImagePullConfig, sf *streamformatter.StreamFormatter) (fallback bool, err error) {
	if err := p.pullV2Repository(image, name, tag, endpoint, imagePullConfig, sf); err != nil {
		if err, ok := err.(*registry.ErrRegistry); ok && err.Fallback {
			return true, err
		}
		logrus.Debugf("Not continuing with error: %v", err)
		return false, err
	}
	return false, nil
}

func (s *TagStore) pullV2Repository(image, name, tag string, endpoint registry.APIEndpoint, imagePullConfig *ImagePullConfig, sf *streamformatter.StreamFormatter) error {
	// TODO(tiborvass): was ReceiveTimeout
	repo, err := NewV2Repository(name, endpoint, imagePullConfig.MetaHeaders, imagePullConfig.AuthConfig)
	if err != nil {
		return registry.WrapError("error creating repository client", err)
	}

	var tags []string
	taggedName := image
	if len(tag) > 1 {
		tags = []string{tag}
		taggedName = utils.ImageReference(image, tag)
	} else {
		var err error
		tags, err = repo.Manifests().Tags()
		if err != nil {
			return registry.WrapError("error getting tags", err)
		}

	}

	c, err := s.poolAdd("pull", taggedName)
	if err != nil {
		if c != nil {
			// Another pull of the same repository is already taking place; just wait for it to finish
			sf.FormatStatus("", "Repository %s already being pulled by another client. Waiting.", image)
			<-c
			return nil
		}
		return err
	}
	defer s.poolRemove("pull", taggedName)

	var pulledNew bool
	for _, tag := range tags {
		pulledNew, err = s.pullV2Tag(repo, image, tag, imagePullConfig.OutStream, sf)
		if err != nil {
			return registry.WrapError("error pulling tags", err)
		}
	}

	WriteStatus(taggedName, imagePullConfig.OutStream, sf, pulledNew)

	return nil
}

func (s *TagStore) pullV2Tag(repo distribution.Repository, localName, tag string, out io.Writer, sf *streamformatter.StreamFormatter) (bool, error) {
	// downloadInfo is used to pass information from download to extractor
	type downloadInfo struct {
		img     *image.Image
		tmpFile *os.File
		digest  digest.Digest
		layer   distribution.ReadSeekCloser
		size    int64
		err     chan error
	}
	var verified bool
	manifest, err := repo.Manifests().GetByTag(tag)
	if err != nil {
		return false, registry.WrapError("error getting image manifest", err)
	}
	// TODO(tiborvass): what's the usecase for having manifest == nil and err == nil ?
	if manifest == nil {
		return false, fmt.Errorf("image manifest does not exist for tag: %s", tag)
	}
	if manifest.SchemaVersion != 1 {
		return false, fmt.Errorf("unsupport image manifest version(%d) for tag: %s", manifest.SchemaVersion, tag)
	}

	out.Write(sf.FormatStatus(tag, "Pulling from %s", repo.Name()))

	downloads := make([]downloadInfo, len(manifest.FSLayers))
	for i := len(manifest.FSLayers) - 1; i >= 0; i-- {
		img, err := image.NewImgJSON([]byte(manifest.History[i].V1Compatibility))
		if err != nil {
			return false, registry.WrapError("error getting image v1 json", err)
		}
		downloads[i].img = img
		downloads[i].digest = manifest.FSLayers[i].BlobSum

		// Check if exists
		if s.graph.Exists(img.ID) {
			logrus.Debugf("Image already exists: %s", img.ID)
			continue
		}

		out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Pulling fs layer", nil))

		downloadFunc := func(di *downloadInfo) error {
			logrus.Debugf("pulling blob %q to %s", di.digest, di.img.ID)

			if c, err := s.poolAdd("pull", "img:"+di.img.ID); err != nil {
				if c != nil {
					out.Write(sf.FormatProgress(stringid.TruncateID(di.img.ID), "Layer already being pulled by another client. Waiting.", nil))
					<-c
					out.Write(sf.FormatProgress(stringid.TruncateID(di.img.ID), "Download complete", nil))
				} else {
					logrus.Debugf("Image (id: %s) pull is already running, skipping: %v", di.img.ID, err)
				}
			} else {
				defer s.poolRemove("pull", "img:"+di.img.ID)
				tmpFile, err := ioutil.TempFile("", "GetImageBlob")
				if err != nil {
					return err
				}

				blobs := repo.Blobs(nil)

				desc, err := blobs.Stat(nil, di.digest)
				if err != nil {
					return registry.WrapError("error statting layer", err)
				}
				di.size = desc.Length

				layerDownload, err := blobs.Open(nil, di.digest)
				if err != nil {
					return registry.WrapError("error fetching layer", err)
				}
				defer layerDownload.Close()

				verifier, err := digest.NewDigestVerifier(di.digest)
				if err != nil {
					return registry.WrapError("creating digest verifier", err)
				}

				reader := progressreader.New(progressreader.Config{
					In:        ioutil.NopCloser(io.TeeReader(layerDownload, verifier)),
					Out:       out,
					Formatter: sf,
					Size:      int(di.size),
					NewLines:  false,
					ID:        stringid.TruncateID(di.img.ID),
					Action:    "Downloading",
				})
				io.Copy(tmpFile, reader)

				out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Verifying Checksum", nil))

				if verifier.Verified() {
					logrus.Infof("Image verification failed for layer %s", di.digest)
					verified = false
				}

				out.Write(sf.FormatProgress(stringid.TruncateID(di.img.ID), "Download complete", nil))

				logrus.Debugf("Downloaded %s to tempfile %s", di.img.ID, tmpFile.Name())
				di.tmpFile = tmpFile
				di.layer = layerDownload
			}

			return nil
		}

		downloads[i].err = make(chan error)
		go func(di *downloadInfo) {
			di.err <- downloadFunc(di)
		}(&downloads[i])

	}

	var layersDownloaded bool
	for i := len(downloads) - 1; i >= 0; i-- {
		d := &downloads[i]
		if d.err != nil {
			if err := <-d.err; err != nil {
				return false, err
			}
		}
		if d.layer != nil {
			// if tmpFile is empty assume download and extracted elsewhere
			defer os.Remove(d.tmpFile.Name())
			defer d.tmpFile.Close()
			d.tmpFile.Seek(0, 0)
			if d.tmpFile != nil {

				reader := progressreader.New(progressreader.Config{
					In:        d.tmpFile,
					Out:       out,
					Formatter: sf,
					Size:      int(d.size),
					NewLines:  false,
					ID:        stringid.TruncateID(d.img.ID),
					Action:    "Extracting",
				})

				err = s.graph.Register(d.img, reader)
				if err != nil {
					return false, err
				}

				// FIXME: Pool release here for parallel tag pull (ensures any downloads block until fully extracted)
			}
			out.Write(sf.FormatProgress(stringid.TruncateID(d.img.ID), "Pull complete", nil))
			layersDownloaded = true
		} else {
			out.Write(sf.FormatProgress(stringid.TruncateID(d.img.ID), "Already exists", nil))
		}
	}

	if utils.DigestReference(tag) {
		if err = s.SetDigest(localName, tag, downloads[0].img.ID); err != nil {
			return false, err
		}
	} else if err = s.Tag(localName, tag, downloads[0].img.ID, true); err != nil {
		return false, err
	}

	if verified && layersDownloaded {
		out.Write(sf.FormatStatus(repo.Name()+":"+tag, "The image you are pulling has been verified. Important: image verification is a tech preview feature and should not be relied on to provide security."))
	}

	manifestDigest, err := digestFromManifest(manifest, localName)
	if err != nil {
		return false, err
	}
	if manifestDigest != "" {
		out.Write(sf.FormatStatus("", "Digest: %s", manifestDigest))
	}

	return layersDownloaded, nil
}
