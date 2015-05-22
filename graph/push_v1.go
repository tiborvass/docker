package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

type v1Pusher struct{ *TagStore }

func (p *v1Pusher) Push(localName, repoName string, endpoint registry.APIEndpoint, imagePushConfig *ImagePushConfig, sf *streamformatter.StreamFormatter) (fallback bool, err error) {
	tlsConfig, err := p.registryService.TlsConfig(repoInfo.Index.Name)
	if err != nil {
		return false, err
	}
	// Adds Docker-specific headers as well as user-specified headers (metaHeaders)
	tr := transport.NewTransport(
		// TODO(tiborvass): was NoTimeout
		registry.NewTransport(tlsConfig),
		registry.DockerHeaders(imagePushConfig.MetaHeaders)...,
	)
	client := registry.HTTPClient(tr)
	v1Endpoint := endpoint.ToV1Endpoint(imagePushConfig.MetaHeaders)
	if v1Endpoint == nil {
		return true, fmt.Errorf("Could not get v1 endpoint")
	}
	r, err := registry.NewSession(client, imagePushConfig.AuthConfig, v1Endpoint)
	if err != nil {
		// TODO(dmcgowan): Check if should fallback
		return true, err
	}
	if err := p.pushRepository(r, imagePushConfig.OutStream, repoInfo, localRepo, imagePushConfig.Tag, sf); err != nil {
		// TODO(dmcgowan): Check if should fallback
		return false, err
	}
	return false, nil
}

// Retrieve the all the images to be uploaded in the correct order
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

// createImageIndex returns an index of an image's layer IDs and tags.
func (s *TagStore) createImageIndex(images []string, tags map[string][]string) []*registry.ImgData {
	var imageIndex []*registry.ImgData
	for _, id := range images {
		if tags, hasTags := tags[id]; hasTags {
			// If an image has tags you must add an entry in the image index
			// for each tag
			for _, tag := range tags {
				imageIndex = append(imageIndex, &registry.ImgData{
					ID:  id,
					Tag: tag,
				})
			}
			continue
		}
		// If the image does not have a tag it still needs to be sent to the
		// registry with an empty tag so that it is accociated with the repository
		imageIndex = append(imageIndex, &registry.ImgData{
			ID:  id,
			Tag: "",
		})
	}
	return imageIndex
}

type imagePushData struct {
	id       string
	endpoint string
	tokens   []string
}

// lookupImageOnEndpoint checks the specified endpoint to see if an image exists
// and if it is absent then it sends the image id to the channel to be pushed.
func lookupImageOnEndpoint(wg *sync.WaitGroup, r *registry.Session, out io.Writer, sf *streamformatter.StreamFormatter,
	images chan imagePushData, imagesToPush chan string) {
	defer wg.Done()
	for image := range images {
		if err := r.LookupRemoteImage(image.id, image.endpoint); err != nil {
			logrus.Errorf("Error in LookupRemoteImage: %s", err)
			imagesToPush <- image.id
			continue
		}
		out.Write(sf.FormatStatus("", "Image %s already pushed, skipping", stringid.TruncateID(image.id)))
	}
}

func (s *TagStore) pushImageToEndpoint(endpoint string, out io.Writer, remoteName string, imageIDs []string,
	tags map[string][]string, repo *registry.RepositoryData, sf *streamformatter.StreamFormatter, r *registry.Session) error {
	workerCount := len(imageIDs)
	// start a maximum of 5 workers to check if images exist on the specified endpoint.
	if workerCount > 5 {
		workerCount = 5
	}
	var (
		wg           = &sync.WaitGroup{}
		imageData    = make(chan imagePushData, workerCount*2)
		imagesToPush = make(chan string, workerCount*2)
		pushes       = make(chan map[string]struct{}, 1)
	)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go lookupImageOnEndpoint(wg, r, out, sf, imageData, imagesToPush)
	}
	// start a go routine that consumes the images to push
	go func() {
		shouldPush := make(map[string]struct{})
		for id := range imagesToPush {
			shouldPush[id] = struct{}{}
		}
		pushes <- shouldPush
	}()
	for _, id := range imageIDs {
		imageData <- imagePushData{
			id:       id,
			endpoint: endpoint,
			tokens:   repo.Tokens,
		}
	}
	// close the channel to notify the workers that there will be no more images to check.
	close(imageData)
	wg.Wait()
	close(imagesToPush)
	// wait for all the images that require pushes to be collected into a consumable map.
	shouldPush := <-pushes
	// finish by pushing any images and tags to the endpoint.  The order that the images are pushed
	// is very important that is why we are still iterating over the ordered list of imageIDs.
	for _, id := range imageIDs {
		if _, push := shouldPush[id]; push {
			if _, err := s.pushImage(r, out, id, endpoint, repo.Tokens, sf); err != nil {
				// FIXME: Continue on error?
				return err
			}
		}
		for _, tag := range tags[id] {
			out.Write(sf.FormatStatus("", "Pushing tag for rev [%s] on {%s}", stringid.TruncateID(id), endpoint+"repositories/"+remoteName+"/tags/"+tag))
			if err := r.PushRegistryTag(remoteName, id, tag, endpoint); err != nil {
				return err
			}
		}
	}
	return nil
}

// pushRepository pushes layers that do not already exist on the registry.
func (s *TagStore) pushRepository(r *registry.Session, out io.Writer,
	repoInfo *registry.RepositoryInfo, localRepo map[string]string,
	tag string, sf *streamformatter.StreamFormatter) error {

	logrus.Debugf("Local repo: %s", localRepo)
	out = ioutils.NewWriteFlusher(out)
	imgList, tags, err := s.getImageList(localRepo, tag)
	if err != nil {
		return err
	}
	out.Write(sf.FormatStatus("", "Sending image list"))

	imageIndex := s.createImageIndex(imgList, tags)
	logrus.Debugf("Preparing to push %s with the following images and tags", localRepo)
	for _, data := range imageIndex {
		logrus.Debugf("Pushing ID: %s with Tag: %s", data.ID, data.Tag)
	}

	if _, err := s.poolAdd("push", repoInfo.LocalName); err != nil {
		return err
	}
	defer s.poolRemove("push", repoInfo.LocalName)

	// Register all the images in a repository with the registry
	// If an image is not in this list it will not be associated with the repository
	repoData, err := r.PushImageJSONIndex(repoInfo.RemoteName, imageIndex, false, nil)
	if err != nil {
		return err
	}
	nTag := 1
	if tag == "" {
		nTag = len(localRepo)
	}
	out.Write(sf.FormatStatus("", "Pushing repository %s (%d tags)", repoInfo.CanonicalName, nTag))
	// push the repository to each of the endpoints only if it does not exist.
	for _, endpoint := range repoData.Endpoints {
		if err := s.pushImageToEndpoint(endpoint, out, repoInfo.RemoteName, imgList, tags, repoData, sf, r); err != nil {
			return err
		}
	}
	_, err = r.PushImageJSONIndex(repoInfo.RemoteName, imageIndex, true, repoData.Endpoints)
	return err
}

func (s *TagStore) pushImage(r *registry.Session, out io.Writer, imgID, ep string, token []string, sf *streamformatter.StreamFormatter) (checksum string, err error) {
	out = ioutils.NewWriteFlusher(out)
	jsonRaw, err := ioutil.ReadFile(filepath.Join(s.graph.Root, imgID, "json"))
	if err != nil {
		return "", fmt.Errorf("Cannot retrieve the path for {%s}: %s", imgID, err)
	}
	out.Write(sf.FormatProgress(stringid.TruncateID(imgID), "Pushing", nil))

	imgData := &registry.ImgData{
		ID: imgID,
	}

	// Send the json
	if err := r.PushImageJSONRegistry(imgData, jsonRaw, ep); err != nil {
		if err == registry.ErrAlreadyExists {
			out.Write(sf.FormatProgress(stringid.TruncateID(imgData.ID), "Image already pushed, skipping", nil))
			return "", nil
		}
		return "", err
	}

	layerData, err := s.graph.TempLayerArchive(imgID, sf, out)
	if err != nil {
		return "", fmt.Errorf("Failed to generate layer archive: %s", err)
	}
	defer os.RemoveAll(layerData.Name())

	// Send the layer
	logrus.Debugf("rendered layer for %s of [%d] size", imgData.ID, layerData.Size)

	checksum, checksumPayload, err := r.PushImageLayerRegistry(imgData.ID,
		progressreader.New(progressreader.Config{
			In:        layerData,
			Out:       out,
			Formatter: sf,
			Size:      int(layerData.Size),
			NewLines:  false,
			ID:        stringid.TruncateID(imgData.ID),
			Action:    "Pushing",
		}), ep, jsonRaw)
	if err != nil {
		return "", err
	}
	imgData.Checksum = checksum
	imgData.ChecksumPayload = checksumPayload
	// Send the checksum
	if err := r.PushImageChecksumRegistry(imgData, ep); err != nil {
		return "", err
	}

	out.Write(sf.FormatProgress(stringid.TruncateID(imgData.ID), "Image successfully pushed", nil))
	return imgData.Checksum, nil
}
