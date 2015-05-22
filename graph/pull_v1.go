package graph

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

type v1Puller struct{ *TagStore }

func (p *v1Puller) Pull(image, name, tag string, endpoint registry.APIEndpoint, imagePullConfig *ImagePullConfig, sf *streamformatter.StreamFormatter) (fallback bool, err error) {
	tlsConfig, err := p.registryService.TlsConfig(repoInfo.Index.Name)
	if err != nil {
		return false, err
	}
	// Adds Docker-specific headers as well as user-specified headers (metaHeaders)
	tr := transport.NewTransport(
		// TODO(tiborvass): was ReceiveTimeout
		registry.NewTransport(tlsConfig),
		registry.DockerHeaders(imagePullConfig.MetaHeaders)...,
	)
	client := registry.HTTPClient(tr)
	v1Endpoint := endpoint.ToV1Endpoint(imagePullConfig.MetaHeaders)
	if v1Endpoint == nil {
		return true, fmt.Errorf("Could not get v1 endpoint")
	}
	r, err := registry.NewSession(client, imagePullConfig.AuthConfig, v1Endpoint)
	if err != nil {
		// TODO(dmcgowan): Check if should fallback
		logrus.Debugf("Fallback from error: %s", err)
		return true, err
	}
	if err := p.pullRepository(r, imagePullConfig.OutStream, repoInfo, tag, sf); err != nil {
		// TODO(dmcgowan): Check if should fallback
		return false, err
	}
	return false, nil
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
