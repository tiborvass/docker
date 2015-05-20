package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

type ImagePullConfig struct {
	MetaHeaders map[string][]string
	AuthConfig  *cliconfig.AuthConfig
	OutStream   io.Writer
}

func (s *TagStore) Pull(image string, tag string, imagePullConfig *ImagePullConfig) error {
	var sf = streamformatter.NewJSONStreamFormatter()

	image = CanonicalizeName(image)

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := s.registryService.ResolveRepository(image)
	if err != nil {
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

		switch endpoint.Version {
		case registry.APIVersion2:
			if err := s.pullV2Repository(image, name, tag, endpoint, imagePullConfig, sf); err != nil {
				if err, ok := err.(*registry.ErrRegistry); ok && err.Fallback {
					lastErr = err
					continue
				}
				logrus.Debugf("Not continuing with error: %v", err)
				return err
			}
		case registry.APIVersion1:
			// TODO(tiborvass): check if endpoint is secure
			secure := false
			// Adds Docker-specific headers as well as user-specified headers (metaHeaders)
			tr := transport.NewTransport(
				registry.NewTransport(registry.ReceiveTimeout, secure),
				registry.DockerHeaders(imagePullConfig.MetaHeaders)...,
			)
			client := registry.HTTPClient(tr)
			v1Endpoint := endpoint.ToV1Endpoint(imagePullConfig.MetaHeaders)
			if v1Endpoint == nil {
				lastErr = fmt.Errorf("Could not get v1 endpoint")
				continue
			}
			r, err := registry.NewSession(client, imagePullConfig.AuthConfig, v1Endpoint)
			if err != nil {
				// TODO(dmcgowan): Check if should fallback
				lastErr = err
				logrus.Debugf("Fallback from error: %s", err)
				continue
			}
			if err := s.pullRepository(r, imagePullConfig.OutStream, repoInfo, tag, sf); err != nil {
				// TODO(dmcgowan): Check if should fallback
				return err
			}
		default:
			lastErr = fmt.Errorf("unknown version %d for registry %s", endpoint.Version, endpoint.URL)
			continue
		}

		s.eventsService.Log("pull", logName, "")
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no endpoints found for %s", image)
	}
	return lastErr
}

func (s *TagStore) pullRepository(r *registry.Session, out io.Writer, repoInfo *registry.RepositoryInfo, askedTag string, sf *streamformatter.StreamFormatter) error {
	out.Write(sf.FormatStatus("", "Pulling repository %s", repoInfo.CanonicalName))

	repoData, err := r.GetRepositoryData(repoInfo.RemoteName)
	if err != nil {
		if strings.Contains(err.Error(), "HTTP code: 404") {
			return fmt.Errorf("Error: image %s not found", utils.ImageReference(repoInfo.RemoteName, askedTag))
		}
		// Unexpected HTTP error
		return err
	}

	logrus.Debugf("Retrieving the tag list")
	tagsList, err := r.GetRemoteTags(repoData.Endpoints, repoInfo.RemoteName)
	if err != nil {
		logrus.Errorf("unable to get remote tags: %s", err)
		return err
	}

	for tag, id := range tagsList {
		repoData.ImgList[id] = &registry.ImgData{
			ID:       id,
			Tag:      tag,
			Checksum: "",
		}
	}

	logrus.Debugf("Registering tags")
	// If no tag has been specified, pull them all
	if askedTag == "" {
		for tag, id := range tagsList {
			repoData.ImgList[id].Tag = tag
		}
	} else {
		// Otherwise, check that the tag exists and use only that one
		id, exists := tagsList[askedTag]
		if !exists {
			return fmt.Errorf("Tag %s not found in repository %s", askedTag, repoInfo.CanonicalName)
		}
		repoData.ImgList[id].Tag = askedTag
	}

	errors := make(chan error)

	layersDownloaded := false
	for _, image := range repoData.ImgList {
		downloadImage := func(img *registry.ImgData) {
			if askedTag != "" && img.Tag != askedTag {
				errors <- nil
				return
			}

			if img.Tag == "" {
				logrus.Debugf("Image (id: %s) present in this repository but untagged, skipping", img.ID)
				errors <- nil
				return
			}

			// ensure no two downloads of the same image happen at the same time
			if c, err := s.poolAdd("pull", "img:"+img.ID); err != nil {
				if c != nil {
					out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Layer already being pulled by another client. Waiting.", nil))
					<-c
					out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Download complete", nil))
				} else {
					logrus.Debugf("Image (id: %s) pull is already running, skipping: %v", img.ID, err)
				}
				errors <- nil
				return
			}
			defer s.poolRemove("pull", "img:"+img.ID)

			out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), fmt.Sprintf("Pulling image (%s) from %s", img.Tag, repoInfo.CanonicalName), nil))
			success := false
			var lastErr, err error
			var isDownloaded bool
			for _, ep := range repoInfo.Index.Mirrors {
				out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), fmt.Sprintf("Pulling image (%s) from %s, mirror: %s", img.Tag, repoInfo.CanonicalName, ep), nil))
				if isDownloaded, err = s.pullImage(r, out, img.ID, ep, repoData.Tokens, sf); err != nil {
					// Don't report errors when pulling from mirrors.
					logrus.Debugf("Error pulling image (%s) from %s, mirror: %s, %s", img.Tag, repoInfo.CanonicalName, ep, err)
					continue
				}
				layersDownloaded = layersDownloaded || isDownloaded
				success = true
				break
			}
			if !success {
				for _, ep := range repoData.Endpoints {
					out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), fmt.Sprintf("Pulling image (%s) from %s, endpoint: %s", img.Tag, repoInfo.CanonicalName, ep), nil))
					if isDownloaded, err = s.pullImage(r, out, img.ID, ep, repoData.Tokens, sf); err != nil {
						// It's not ideal that only the last error is returned, it would be better to concatenate the errors.
						// As the error is also given to the output stream the user will see the error.
						lastErr = err
						out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), fmt.Sprintf("Error pulling image (%s) from %s, endpoint: %s, %s", img.Tag, repoInfo.CanonicalName, ep, err), nil))
						continue
					}
					layersDownloaded = layersDownloaded || isDownloaded
					success = true
					break
				}
			}
			if !success {
				err := fmt.Errorf("Error pulling image (%s) from %s, %v", img.Tag, repoInfo.CanonicalName, lastErr)
				out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), err.Error(), nil))
				errors <- err
				return
			}
			out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Download complete", nil))

			errors <- nil
		}

		go downloadImage(image)
	}

	var lastError error
	for i := 0; i < len(repoData.ImgList); i++ {
		if err := <-errors; err != nil {
			lastError = err
		}
	}
	if lastError != nil {
		return lastError
	}

	for tag, id := range tagsList {
		if askedTag != "" && tag != askedTag {
			continue
		}
		if err := s.Tag(repoInfo.LocalName, tag, id, true); err != nil {
			return err
		}
	}

	requestedTag := repoInfo.CanonicalName
	if len(askedTag) > 0 {
		requestedTag = utils.ImageReference(repoInfo.CanonicalName, askedTag)
	}
	WriteStatus(requestedTag, out, sf, layersDownloaded)
	return nil
}

func (s *TagStore) pullImage(r *registry.Session, out io.Writer, imgID, endpoint string, token []string, sf *streamformatter.StreamFormatter) (bool, error) {
	history, err := r.GetRemoteHistory(imgID, endpoint)
	if err != nil {
		return false, err
	}
	out.Write(sf.FormatProgress(stringid.TruncateID(imgID), "Pulling dependent layers", nil))
	// FIXME: Try to stream the images?
	// FIXME: Launch the getRemoteImage() in goroutines

	layersDownloaded := false
	for i := len(history) - 1; i >= 0; i-- {
		id := history[i]

		// ensure no two downloads of the same layer happen at the same time
		if c, err := s.poolAdd("pull", "layer:"+id); err != nil {
			logrus.Debugf("Image (id: %s) pull is already running, skipping: %v", id, err)
			<-c
		}
		defer s.poolRemove("pull", "layer:"+id)

		if !s.graph.Exists(id) {
			out.Write(sf.FormatProgress(stringid.TruncateID(id), "Pulling metadata", nil))
			var (
				imgJSON []byte
				imgSize int
				err     error
				img     *image.Image
			)
			retries := 5
			for j := 1; j <= retries; j++ {
				imgJSON, imgSize, err = r.GetRemoteImageJSON(id, endpoint)
				if err != nil && j == retries {
					out.Write(sf.FormatProgress(stringid.TruncateID(id), "Error pulling dependent layers", nil))
					return layersDownloaded, err
				} else if err != nil {
					time.Sleep(time.Duration(j) * 500 * time.Millisecond)
					continue
				}
				img, err = image.NewImgJSON(imgJSON)
				layersDownloaded = true
				if err != nil && j == retries {
					out.Write(sf.FormatProgress(stringid.TruncateID(id), "Error pulling dependent layers", nil))
					return layersDownloaded, fmt.Errorf("Failed to parse json: %s", err)
				} else if err != nil {
					time.Sleep(time.Duration(j) * 500 * time.Millisecond)
					continue
				} else {
					break
				}
			}

			for j := 1; j <= retries; j++ {
				// Get the layer
				status := "Pulling fs layer"
				if j > 1 {
					status = fmt.Sprintf("Pulling fs layer [retries: %d]", j)
				}
				out.Write(sf.FormatProgress(stringid.TruncateID(id), status, nil))
				layer, err := r.GetRemoteImageLayer(img.ID, endpoint, int64(imgSize))
				if uerr, ok := err.(*url.Error); ok {
					err = uerr.Err
				}
				if terr, ok := err.(net.Error); ok && terr.Timeout() && j < retries {
					time.Sleep(time.Duration(j) * 500 * time.Millisecond)
					continue
				} else if err != nil {
					out.Write(sf.FormatProgress(stringid.TruncateID(id), "Error pulling dependent layers", nil))
					return layersDownloaded, err
				}
				layersDownloaded = true
				defer layer.Close()

				err = s.graph.Register(img,
					progressreader.New(progressreader.Config{
						In:        layer,
						Out:       out,
						Formatter: sf,
						Size:      imgSize,
						NewLines:  false,
						ID:        stringid.TruncateID(id),
						Action:    "Downloading",
					}))
				if terr, ok := err.(net.Error); ok && terr.Timeout() && j < retries {
					time.Sleep(time.Duration(j) * 500 * time.Millisecond)
					continue
				} else if err != nil {
					out.Write(sf.FormatProgress(stringid.TruncateID(id), "Error downloading dependent layers", nil))
					return layersDownloaded, err
				} else {
					break
				}
			}
		}
		out.Write(sf.FormatProgress(stringid.TruncateID(id), "Download complete", nil))
	}
	return layersDownloaded, nil
}

func WriteStatus(requestedTag string, out io.Writer, sf *streamformatter.StreamFormatter, layersDownloaded bool) {
	if layersDownloaded {
		out.Write(sf.FormatStatus("", "Status: Downloaded newer image for %s", requestedTag))
	} else {
		out.Write(sf.FormatStatus("", "Status: Image is up to date for %s", requestedTag))
	}
}

func (s *TagStore) pullV2Repository(image, name, tag string, endpoint registry.APIEndpoint, imagePullConfig *ImagePullConfig, sf *streamformatter.StreamFormatter) error {
	/*
		TODO(tiborvass): make sure it behaves the same as the old code below

		endpoint, err := repoInfo.GetEndpoint(imagePullConfig.MetaHeaders)
		if err != nil {
			return err
		}
		// TODO(tiborvass): reuse client from endpoint?
		// Adds Docker-specific headers as well as user-specified headers (metaHeaders)
		tr := transport.NewTransport(
			registry.NewTransport(registry.ReceiveTimeout, endpoint.IsSecure),
			registry.DockerHeaders(imagePullConfig.MetaHeaders)...,
		)
		client := registry.HTTPClient(tr)
		r, err := registry.NewSession(client, imagePullConfig.AuthConfig, endpoint)
	*/
	// TODO(dmcgowan): Pass tls configuration
	repo, err := NewV2Repository(name, endpoint, imagePullConfig.MetaHeaders, imagePullConfig.AuthConfig)
	if err != nil {
		return registry.WrapError("error creating repository client", err)
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

	downloads := make([]downloadInfo, len(manifest.FSLayers))
	for i := len(manifest.FSLayers) - 1; i >= 0; i-- {
		img, err := image.NewImgJSON([]byte(manifest.History[i].V1Compatibility))
		if err != nil {
			return false, registry.WrapError("error getting image manifest", err)
		}
		if manifest == nil {
			return false, fmt.Errorf("image manifest does not exist for tag", tag)
		}
		if manifest.SchemaVersion != 1 {
			return false, fmt.Errorf("unsupported image manifest version(%d) for tag: %s", manifest.SchemaVersion, tag)
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

				/*

					// TODO(tiborvass): does any part of the code need to do this
					// seek dance to retrieve size?

					if size, err := layerDownload.Seek(0, os.SEEK_END); err != nil {
						return fmt.Errorf("error seeking to end: %v", err)
					} else if size == 0 {
						return fmt.Errorf("layer did not return a size: %s", di.digest)
					} else {
						di.size = size
					}
					if _, err := layerDownload.Seek(0, 0); err != nil {
						return fmt.Errorf("error seeking to beginning: %v", err)
					}
				*/

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

	if verified && layersDownloaded {
		out.Write(sf.FormatStatus(repo.Name()+":"+tag, "The image you are pulling has been verified. Important: image verification is a tech preview feature and should not be relied on to provide security."))
	}

	return layersDownloaded, nil
}
