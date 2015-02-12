package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
)

type ImagePullConfig struct {
	Parallel    bool
	MetaHeaders map[string][]string
	AuthConfig  *cliconfig.AuthConfig
	Json        bool
	OutStream   io.Writer
}

func (s *TagStore) Pull(image string, tag string, imagePullConfig *ImagePullConfig) error {
	var (
		sf = streamformatter.NewStreamFormatter(imagePullConfig.Json)
	)

	repo, err := NewRepositoryClient(CanonicalizeName(image), imagePullConfig.MetaHeaders, imagePullConfig.AuthConfig)
	if err != nil {
		return err
	}

	var tags []string
	taggedName := image
	if len(tag) > 1 {
		tags = []string{tag}
		taggedName = image + ":" + tag
	} else {
		var err error
		tags, err = repo.Manifests().Tags()
		if err != nil {
			return fmt.Errorf("error getting tags: %s", err)
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

	pulledNew, err := s.pullTags(repo, image, tags, imagePullConfig.OutStream, sf)
	if err != nil {
		return err
	}

	WriteStatus(taggedName, imagePullConfig.OutStream, sf, pulledNew)

	s.eventsService.Log("pull", taggedName, "")

	return nil
}

func (s *TagStore) pullTags(repo distribution.Repository, localName string, tags []string, out io.Writer, sf *streamformatter.StreamFormatter) (bool, error) {
	var newPullLayers bool

	// downloadInfo is used to pass information from download to extractor
	type downloadInfo struct {
		img     *image.Image
		tmpFile *os.File
		digest  digest.Digest
		layer   distribution.Layer
		size    int64
		err     chan error
	}

	for _, tag := range tags {
		var verified bool
		manifest, err := repo.Manifests().GetByTag(tag)
		if err != nil {
			return false, fmt.Errorf("error getting image manifest: %s", err)
		}
		if manifest == nil {
			return false, fmt.Errorf("image manifest does not exist for tag: %s", tag)
		}
		if manifest.SchemaVersion != 1 {
			return false, fmt.Errorf("unsupport image manifest version(%d) for tag: %s", manifest.SchemaVersion, tag)
		}

		downloads := make([]downloadInfo, len(manifest.FSLayers))
		for i := len(manifest.FSLayers) - 1; i >= 0; i-- {
			img, err := image.NewImgJSON([]byte(manifest.History[i].V1Compatibility))
			if err != nil {
				return false, fmt.Errorf("failed to parse json: %s", err)
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
				logrus.Debugf("pulling blob %q to %s", di.digest, img.ID)

				if c, err := s.poolAdd("pull", "img:"+img.ID); err != nil {
					if c != nil {
						out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Layer already being pulled by another client. Waiting.", nil))
						<-c
						out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Download complete", nil))
					} else {
						logrus.Debugf("Image (id: %s) pull is already running, skipping: %v", img.ID, err)
					}
				} else {
					defer s.poolRemove("pull", "img:"+img.ID)
					tmpFile, err := ioutil.TempFile("", "GetImageBlob")
					if err != nil {
						return err
					}

					layerDownload, err := repo.Layers().Fetch(di.digest)
					if err != nil {
						return fmt.Errorf("error fetching layer: %s", err)
					}
					defer layerDownload.Close()

					if size, err := layerDownload.Seek(0, os.SEEK_END); err != nil {
						return fmt.Errorf("error seeking to end: %s", err)
					} else if size == 0 {
						return fmt.Errorf("layer did not return a size: %s", di.digest)
					} else {
						di.size = size
					}
					if _, err := layerDownload.Seek(0, 0); err != nil {
						return fmt.Errorf("error seeking to beginning: %s", err)
					}

					verifier, err := digest.NewDigestVerifier(layerDownload.Digest())
					if err != nil {
						return err
					}

					reader := progressreader.New(progressreader.Config{
						In:        ioutil.NopCloser(io.TeeReader(layerDownload, verifier)),
						Out:       out,
						Formatter: sf,
						Size:      int(di.size),
						NewLines:  false,
						ID:        stringid.TruncateID(img.ID),
						Action:    "Downloading",
					})
					io.Copy(tmpFile, reader)

					out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Verifying Checksum", nil))

					if verifier.Verified() {
						logrus.Infof("Image verification failed for layer %s", di.digest)
						verified = false
					}

					out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Download complete", nil))

					logrus.Debugf("Downloaded %s to tempfile %s", img.ID, tmpFile.Name())
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
				err := <-d.err
				if err != nil {
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

		if err = s.Tag(localName, tag, downloads[0].img.ID, true); err != nil {
			return false, err
		}

		if layersDownloaded {
			newPullLayers = true
		}
		if verified && layersDownloaded {
			out.Write(sf.FormatStatus(repo.Name()+":"+tag, "The image you are pulling has been verified. Important: image verification is a tech preview feature and should not be relied on to provide security."))
		}
	}

	return newPullLayers, nil
}

func WriteStatus(requestedTag string, out io.Writer, sf *streamformatter.StreamFormatter, layersDownloaded bool) {
	if layersDownloaded {
		out.Write(sf.FormatStatus("", "Status: Downloaded newer image for %s", requestedTag))
	} else {
		out.Write(sf.FormatStatus("", "Status: Image is up to date for %s", requestedTag))
	}
}
