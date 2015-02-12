package graph

import (
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

type ImagePullConfig struct {
	MetaHeaders map[string][]string
	AuthConfig  *cliconfig.AuthConfig
	OutStream   io.Writer
}

type Puller interface {
	// Pull tries to pull the image referenced by `tag`
	// Pull returns an error if any, as well as a boolean that determines whether to retry Pull on the next configured endpoint.
	//
	// TODO(tiborvass): have Pull() take a reference to repository + tag, so that the puller itself is repository-agnostic.
	Pull(tag string) (fallback bool, err error)
}

func NewPuller(s *TagStore, endpoint registry.APIEndpoint, repoInfo *registry.RepositoryInfo, imagePullConfig *ImagePullConfig, sf *streamformatter.StreamFormatter) (Puller, error) {
	switch endpoint.Version {
	case registry.APIVersion2:
		return &v2Puller{
			TagStore: s,
			endpoint: endpoint,
			config:   imagePullConfig,
			sf:       sf,
			repoInfo: repoInfo,
		}, nil
	case registry.APIVersion1:
		return &v1Puller{
			TagStore: s,
			endpoint: endpoint,
			config:   imagePullConfig,
			sf:       sf,
			repoInfo: repoInfo,
		}, nil
	}
	return nil, fmt.Errorf("unknown version %d for registry %s", endpoint.Version, endpoint.URL)
}

func (s *TagStore) Pull(image string, tag string, imagePullConfig *ImagePullConfig) error {
	var sf = streamformatter.NewJSONStreamFormatter()

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := s.registryService.ResolveRepository(image)
	if err != nil {
		return err
	}

	// makes sure name is not empty or `scratch`
	if err := validateRepoName(repoInfo.LocalName); err != nil {
		return err
	}

	endpoints, err := s.registryService.LookupEndpoints(repoInfo.CanonicalName)
	if err != nil {
		return err
	}

	logName := repoInfo.LocalName
	if tag != "" {
		logName = utils.ImageReference(logName, tag)
	}

	var (
		lastErr error

		// discardNoSupportErrors is used to track whether an endpoint encountered an error of type registry.ErrNoSupport
		// By default it is false, which means that if a ErrNoSupport error is encountered, it will be saved in lastErr.
		// As soon as another kind of error is encountered, discardNoSupportErrors is set to true, avoiding the saving of
		// any subsequent ErrNoSupport errors in lastErr.
		// It's needed for pull-by-digest on v1 endpoints: if there are only v1 endpoints configured, the error should be
		// returned and displayed, but if there was a v2 endpoint which supports pull-by-digest, then the last relevant
		// error is the ones from v2 endpoints not v1.
		discardNoSupportErrors bool
	)
	for _, endpoint := range endpoints {
		logrus.Debugf("Trying to pull %s from %s %s", repoInfo.LocalName, endpoint.URL, endpoint.Version)

		if !endpoint.Mirror && (endpoint.Official || endpoint.Version == registry.APIVersion2) {
			if repoInfo.Official {
				s.trustService.UpdateBase()
			}
		}

		puller, err := NewPuller(s, endpoint, repoInfo, imagePullConfig, sf)
		if err != nil {
			lastErr = err
			continue
		}
		if fallback, err := puller.Pull(tag); err != nil {
			if fallback {
				if _, ok := err.(registry.ErrNoSupport); !ok {
					// Because we found an error that's not ErrNoSupport, discard all subsequent ErrNoSupport errors.
					discardNoSupportErrors = true
					// save the current error
					lastErr = err
				} else if !discardNoSupportErrors {
					// Save the ErrNoSupport error, because it's either the first error or all encountered errors
					// were also ErrNoSupport errors.
					lastErr = err
				}
				continue
			}
			logrus.Debugf("Not continuing with error: %v", err)
			return err

		}

		s.eventsService.Log("pull", logName, "")
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no endpoints found for %s", image)
	}
	return lastErr
}

func WriteStatus(requestedTag string, out io.Writer, sf *streamformatter.StreamFormatter, layersDownloaded bool) {
	if layersDownloaded {
		out.Write(sf.FormatStatus("", "Status: Downloaded newer image for %s", requestedTag))
	} else {
		out.Write(sf.FormatStatus("", "Status: Image is up to date for %s", requestedTag))
	}
}

func (s *TagStore) pullV2Repository(r *registry.Session, out io.Writer, repoInfo *registry.RepositoryInfo, tag string, sf *streamformatter.StreamFormatter) error {
	endpoint, err := r.V2RegistryEndpoint(repoInfo.Index)
	if err != nil {
		if repoInfo.Index.Official {
			logrus.Debugf("Unable to pull from V2 registry, falling back to v1: %s", err)
			return ErrV2RegistryUnavailable
		}
		return fmt.Errorf("error getting registry endpoint: %s", err)
	}
	auth, err := r.GetV2Authorization(endpoint, repoInfo.RemoteName, true)
	if err != nil {
		return fmt.Errorf("error getting authorization: %s", err)
	}
	if !auth.CanAuthorizeV2() {
		return ErrV2RegistryUnavailable
	}

	var layersDownloaded bool
	if tag == "" {
		logrus.Debugf("Pulling tag list from V2 registry for %s", repoInfo.CanonicalName)
		tags, err := r.GetV2RemoteTags(endpoint, repoInfo.RemoteName, auth)
		if err != nil {
			return err
		}
		if len(tags) == 0 {
			return registry.ErrDoesNotExist
		}
		for _, t := range tags {
			if downloaded, err := s.pullV2Tag(r, out, endpoint, repoInfo, t, sf, auth); err != nil {
				return err
			} else if downloaded {
				layersDownloaded = true
			}
		}
	} else {
		if downloaded, err := s.pullV2Tag(r, out, endpoint, repoInfo, tag, sf, auth); err != nil {
			return err
		} else if downloaded {
			layersDownloaded = true
		}
	}

	requestedTag := repoInfo.LocalName
	if len(tag) > 0 {
		requestedTag = utils.ImageReference(repoInfo.LocalName, tag)
	}
	WriteStatus(requestedTag, out, sf, layersDownloaded)
	return nil
}

func (s *TagStore) pullV2Tag(r *registry.Session, out io.Writer, endpoint *registry.Endpoint, repoInfo *registry.RepositoryInfo, tag string, sf *streamformatter.StreamFormatter, auth *registry.RequestAuthorization) (bool, error) {
	logrus.Debugf("Pulling tag from V2 registry: %q", tag)

	remoteDigest, manifestBytes, err := r.GetV2ImageManifest(endpoint, repoInfo.RemoteName, tag, auth)
	if err != nil {
		return false, err
	}

	// loadManifest ensures that the manifest payload has the expected digest
	// if the tag is a digest reference.
	localDigest, manifest, verified, err := s.loadManifest(manifestBytes, tag, remoteDigest)
	if err != nil {
		return false, fmt.Errorf("error verifying manifest: %s", err)
	}

	if verified {
		logrus.Printf("Image manifest for %s has been verified", utils.ImageReference(repoInfo.CanonicalName, tag))
	}
	out.Write(sf.FormatStatus(tag, "Pulling from %s", repoInfo.CanonicalName))

	// downloadInfo is used to pass information from download to extractor
	type downloadInfo struct {
		imgJSON    []byte
		img        *Image
		digest     digest.Digest
		tmpFile    *os.File
		length     int64
		downloaded bool
		err        chan error
	}

	downloads := make([]downloadInfo, len(manifest.FSLayers))

	for i := len(manifest.FSLayers) - 1; i >= 0; i-- {
		var (
			sumStr  = manifest.FSLayers[i].BlobSum
			imgJSON = []byte(manifest.History[i].V1Compatibility)
		)

		img, err := NewImgJSON(imgJSON)
		if err != nil {
			return false, fmt.Errorf("failed to parse json: %s", err)
		}
		downloads[i].img = img

		// Check if exists
		if s.graph.Exists(img.ID) {
			logrus.Debugf("Image already exists: %s", img.ID)
			continue
		}

		dgst, err := digest.ParseDigest(sumStr)
		if err != nil {
			return false, err
		}
		downloads[i].digest = dgst

		out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Pulling fs layer", nil))

		downloadFunc := func(di *downloadInfo) error {
			logrus.Debugf("pulling blob %q to V1 img %s", sumStr, img.ID)

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
				tmpFile, err := ioutil.TempFile("", "GetV2ImageBlob")
				if err != nil {
					return err
				}

				r, l, err := r.GetV2ImageBlobReader(endpoint, repoInfo.RemoteName, di.digest, auth)
				if err != nil {
					return err
				}
				defer r.Close()

				verifier, err := digest.NewDigestVerifier(di.digest)
				if err != nil {
					return err
				}

				if _, err := io.Copy(tmpFile, progressreader.New(progressreader.Config{
					In:        ioutil.NopCloser(io.TeeReader(r, verifier)),
					Out:       out,
					Formatter: sf,
					Size:      int(l),
					NewLines:  false,
					ID:        stringid.TruncateID(img.ID),
					Action:    "Downloading",
				})); err != nil {
					return fmt.Errorf("unable to copy v2 image blob data: %s", err)
				}

				out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Verifying Checksum", nil))

				if !verifier.Verified() {
					return fmt.Errorf("image layer digest verification failed for %q", di.digest)
				}

				out.Write(sf.FormatProgress(stringid.TruncateID(img.ID), "Download complete", nil))

				logrus.Debugf("Downloaded %s to tempfile %s", img.ID, tmpFile.Name())
				di.tmpFile = tmpFile
				di.length = l
				di.downloaded = true
			}
			di.imgJSON = imgJSON

			return nil
		}

		downloads[i].err = make(chan error)
		go func(di *downloadInfo) {
			di.err <- downloadFunc(di)
		}(&downloads[i])
	}

	var tagUpdated bool
	for i := len(downloads) - 1; i >= 0; i-- {
		d := &downloads[i]
		if d.err != nil {
			if err := <-d.err; err != nil {
				return false, err
			}
		}
		if d.downloaded {
			// if tmpFile is empty assume download and extracted elsewhere
			defer os.Remove(d.tmpFile.Name())
			defer d.tmpFile.Close()
			d.tmpFile.Seek(0, 0)
			if d.tmpFile != nil {
				err = s.graph.Register(d.img,
					progressreader.New(progressreader.Config{
						In:        d.tmpFile,
						Out:       out,
						Formatter: sf,
						Size:      int(d.length),
						ID:        stringid.TruncateID(d.img.ID),
						Action:    "Extracting",
					}))
				if err != nil {
					return false, err
				}

				if err := d.img.SaveCheckSum(s.graph.ImageRoot(d.img.ID), d.digest.String()); err != nil {
					return false, err
				}

				// FIXME: Pool release here for parallel tag pull (ensures any downloads block until fully extracted)
			}
			out.Write(sf.FormatProgress(stringid.TruncateID(d.img.ID), "Pull complete", nil))
			tagUpdated = true
		} else {
			out.Write(sf.FormatProgress(stringid.TruncateID(d.img.ID), "Already exists", nil))
		}

	}

	// Check for new tag if no layers downloaded
	if !tagUpdated {
		repo, err := s.Get(repoInfo.LocalName)
		if err != nil {
			return false, err
		}
		if repo != nil {
			if _, exists := repo[tag]; !exists {
				tagUpdated = true
			}
		} else {
			tagUpdated = true
		}
	}

	if verified && tagUpdated {
		out.Write(sf.FormatStatus(utils.ImageReference(repoInfo.CanonicalName, tag), "The image you are pulling has been verified. Important: image verification is a tech preview feature and should not be relied on to provide security."))
	}

	if localDigest != remoteDigest { // this is not a verification check.
		// NOTE(stevvooe): This is a very defensive branch and should never
		// happen, since all manifest digest implementations use the same
		// algorithm.
		logrus.WithFields(
			logrus.Fields{
				"local":  localDigest,
				"remote": remoteDigest,
			}).Debugf("local digest does not match remote")

		out.Write(sf.FormatStatus("", "Remote Digest: %s", remoteDigest))
	}

	out.Write(sf.FormatStatus("", "Digest: %s", localDigest))

	if tag == localDigest.String() {
		// TODO(stevvooe): Ideally, we should always set the digest so we can
		// use the digest whether we pull by it or not. Unfortunately, the tag
		// store treats the digest as a separate tag, meaning there may be an
		// untagged digest image that would seem to be dangling by a user.

		if err = s.SetDigest(repoInfo.LocalName, localDigest.String(), downloads[0].img.ID); err != nil {
			return false, err
		}
	}

	if !utils.DigestReference(tag) {
		// only set the repository/tag -> image ID mapping when pulling by tag (i.e. not by digest)
		if err = s.Tag(repoInfo.LocalName, tag, downloads[0].img.ID, true); err != nil {
			return false, err
		}
	}

	return tagUpdated, nil
}
