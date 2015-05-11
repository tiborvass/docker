package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/digest"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/registry"
)

type ImagePullConfig struct {
	Parallel    bool
	MetaHeaders map[string][]string
	AuthConfig  *cliconfig.AuthConfig
	Json        bool
	OutStream   io.Writer
}

func (s *TagStore) Pull(repoName, tag string, imagePullConfig *ImagePullConfig) error {
	sf := streamformatter.NewStreamFormatter(imagePullConfig.Json)

	// TODO: canonicalize on the client
	repoName = registry.CanonicalizeName(repoName)

	// TODO: dont use a string for action "pull"
	repo, err := s.registryService.NewRepository(repoName, "pull", imagePullConfig.MetaHeaders, imagePullConfig.AuthConfig)
	if err != nil {
		return err
	}

	// pull all tags if tag was not specified, but only download specified tag otherwise
	// TODO: why len(tag) > 1 ? why not > 0 ?
	var tags []string
	taggedName := repoName
	if len(tag) > 1 {
		tags = []string{tag}
		taggedName = repoName + ":" + tag
	} else {
		// Tags(), being the first method needing to contact a registry,
		// will try sequentially the different endpoints defined in NewRepository
		// taking into account settings for insecure registries and mirrors.
		//
		// A list of endpoints are saved for later use.
		//
		// The v1 implementation of Tags() will save the endpoints specified by
		// the X-Docker-Endpoints header in the HTTP response. On a subsequent request,
		// each endpoint of that list will be tried sequentially until one succeeds.
		//
		// v1 at this point already knows the IDs where the tags point to, so it will be cached
		// so that later on Layers(tag) can call GetRemoteHistory(ID) in v1
		tags, err = repo.Tags()
		if err != nil {
			return fmt.Errorf("error getting tags: %v", err)
		}

	}

	// TODO: why replace utils.ImageReference(localname, tag) by taggedName?
	c, err := s.poolAdd("pull", taggedName)
	if err != nil {
		if c != nil {
			// Another pull of the same repository is already taking place; just wait for it to finish
			//TODO: why remove imagePullConfig.OutStream.Write(sf....) ?
			sf.FormatStatus("", "Repository %s already being pulled by another client. Waiting.", repoName)
			<-c
			return nil
		}
		return err
	}
	defer s.poolRemove("pull", taggedName)

	pulledNew, err := s.pullTags(repo, tags, imagePullConfig.OutStream, sf)
	if err != nil {
		return err
	}

	WriteStatus(taggedName, imagePullConfig.OutStream, sf, pulledNew)

	s.eventsService.Log("pull", taggedName, "")

	return nil
}

func (s *TagStore) pullTags(repo registry.Repository, tags []string, out io.Writer, sf *streamformatter.StreamFormatter) (bool, error) {
	var newPullLayers bool

	repoName := repo.Name()

	// downloadInfo is used to pass information from download to extractor
	type downloadInfo struct {
		img     *image.Image
		tmpFile *os.File
		digest  digest.Digest
		size    int64
		errCh   chan error
	}

	for _, tag := range tags {
		// Download metadata for a particular image (tag on a repo)
		layers, err := repo.Layers(tag)
		if err != nil {
			// layers for specified tag not found, probably because tag does not exist for that repo
			return false, err
		}

		out.Write(sf.FormatStatus(tag, "Pulling from %s", repoName))

		downloads := make([]downloadInfo, len(layers))

		var verified bool

		// start downloading each layer in parallel
		for i := len(layers) - 1; i >= 0; i-- {
			json, err := layers[i].V1Json()
			if err != nil {
				return false, err
			}
			img, err := image.NewImageJSON(json)
			if err != nil {
				return false, fmt.Errorf("failed to parse json: %v", err)
			}

			downloads[i] = downloadInfo{
				img:    img,
				digest: layers[i].Digest(), // empty for v1
			}

			shortId := stringid.TruncateID(img.ID)

			// Check if exists
			if s.graph.Exists(img.ID) {
				logrus.Debugf("Image already exists: %s", img.ID)
				out.Write(sf.FormatProgress(shortId, "Already exists", nil))
				continue
			}

			out.Write(sf.FormatProgress(shortId, "Pulling fs layer", nil))

			downloadFunc := func(di *downloadInfo) error {
				logrus.Debugf("pulling blob %q to %s", di.digest, img.ID)

				// ensure no two downloads of the same layer happen at the same time
				if c, err := s.poolAdd("pull", "img:"+img.ID); err != nil {
					if c != nil {
						out.Write(sf.FormatProgress(shortId, "Layer already being pulled by another client. Waiting.", nil))
						<-c
						out.Write(sf.FormatProgress(shortId, "Download complete", nil))
					} else {
						logrus.Debugf("Image (id: %s) pull is already running, skipping: %v", img.ID, err)
					}
					return nil
				}
				defer s.poolRemove("pull", "img:"+img.ID)

				tmpFile, err := ioutil.TempFile("", "GetImageBlob")
				if err != nil {
					return err
				}

				// Fetch will go through the set of endpoints defined by the first call to Tags(), in order to get the raw layers (blobs).
				// If v1 mirrors were set, they are tried first, and then fallback to the other endpoints.
				layerDownload, size, verify, err := layers[i].Fetch()
				if err != nil {
					return err
				}
				defer layerDownload.Close()

				di.size = size

				reader := progressreader.New(progressreader.Config{
					In:        layerDownload,
					Out:       out,
					Formatter: sf,
					Size:      int(di.size),
					NewLines:  false,
					ID:        shortId,
					Action:    "Downloading",
				})

				// TODO: what to do if io.Copy fails?
				io.Copy(tmpFile, reader)

				if verify != nil {
					out.Write(sf.FormatProgress(shortId, "Verifying Checksum", nil))
					verified = verify()
					if !verified {
						logrus.Infof("Image verification failed: checksum mismatch for %s", di.digest)
					}
				}

				out.Write(sf.FormatProgress(shortId, "Download complete", nil))

				logrus.Debugf("Downloaded %s to tempfile %s", img.ID, tmpFile.Name())
				// setting tmpFile is also a way of signaling that this layer was downloaded successfully.
				di.tmpFile = tmpFile

				return nil
			}

			downloads[i].errCh = make(chan error)
			go func(di *downloadInfo) {
				di.errCh <- downloadFunc(di)
			}(&downloads[i])

		}

		var layersDownloaded bool

		// Extract each downloaded layer in order
		for i := len(downloads) - 1; i >= 0; i-- {
			d := &downloads[i]

			// A nil errCh means that nothing was downloaded
			if d.errCh == nil {
				continue
			}

			// blocks until the layer is downloaded
			if err := <-d.errCh; err != nil {
				return false, err
			}

			shortId := stringid.TruncateID(d.img.ID)

			defer os.Remove(d.tmpFile.Name())
			defer d.tmpFile.Close()
			d.tmpFile.Seek(0, 0)

			reader := progressreader.New(progressreader.Config{
				In:        d.tmpFile,
				Out:       out,
				Formatter: sf,
				Size:      int(d.size),
				NewLines:  false,
				ID:        shortId,
				Action:    "Extracting",
			})

			err = s.graph.Register(d.img, reader)
			if err != nil {
				return false, err
			}

			// FIXME: Pool release here for parallel tag pull (ensures any downloads block until fully extracted)

			out.Write(sf.FormatProgress(shortId, "Pull complete", nil))
			layersDownloaded = true
		}

		if err := s.Tag(repoName, tag, downloads[0].img.ID, true); err != nil {
			return false, err
		}

		if layersDownloaded {
			newPullLayers = true
			if verified {
				out.Write(sf.FormatStatus(repoName+":"+tag, "The image you are pulling has been verified. Important: image verification is a tech preview feature and should not be relied on to provide security."))
			}
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
