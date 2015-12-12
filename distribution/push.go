package distribution

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/daemon/events"
	"github.com/tiborvass/docker/distribution/metadata"
	"github.com/tiborvass/docker/distribution/xfer"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/layer"
	"github.com/tiborvass/docker/pkg/progress"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/tag"
	"github.com/docker/libtrust"
	"golang.org/x/net/context"
)

// ImagePushConfig stores push configuration.
type ImagePushConfig struct {
	// MetaHeaders store HTTP headers with metadata about the image
	// (DockerHeaders with prefix X-Meta- in the request).
	MetaHeaders map[string][]string
	// AuthConfig holds authentication credentials for authenticating with
	// the registry.
	AuthConfig *types.AuthConfig
	// ProgressOutput is the interface for showing the status of the push
	// operation.
	ProgressOutput progress.Output
	// RegistryService is the registry service to use for TLS configuration
	// and endpoint lookup.
	RegistryService *registry.Service
	// EventsService is the events service to use for logging.
	EventsService *events.Events
	// MetadataStore is the storage backend for distribution-specific
	// metadata.
	MetadataStore metadata.Store
	// LayerStore manages layers.
	LayerStore layer.Store
	// ImageStore manages images.
	ImageStore image.Store
	// TagStore manages tags.
	TagStore tag.Store
	// TrustKey is the private key for legacy signatures. This is typically
	// an ephemeral key, since these signatures are no longer verified.
	TrustKey libtrust.PrivateKey
	// UploadManager dispatches uploads.
	UploadManager *xfer.LayerUploadManager
}

// Pusher is an interface that abstracts pushing for different API versions.
type Pusher interface {
	// Push tries to push the image configured at the creation of Pusher.
	// Push returns an error if any, as well as a boolean that determines whether to retry Push on the next configured endpoint.
	//
	// TODO(tiborvass): have Push() take a reference to repository + tag, so that the pusher itself is repository-agnostic.
	Push(ctx context.Context) (fallback bool, err error)
}

const compressionBufSize = 32768

// NewPusher creates a new Pusher interface that will push to either a v1 or v2
// registry. The endpoint argument contains a Version field that determines
// whether a v1 or v2 pusher will be created. The other parameters are passed
// through to the underlying pusher implementation for use during the actual
// push operation.
func NewPusher(ref reference.Named, endpoint registry.APIEndpoint, repoInfo *registry.RepositoryInfo, imagePushConfig *ImagePushConfig) (Pusher, error) {
	switch endpoint.Version {
	case registry.APIVersion2:
		return &v2Pusher{
			blobSumService: metadata.NewBlobSumService(imagePushConfig.MetadataStore),
			ref:            ref,
			endpoint:       endpoint,
			repoInfo:       repoInfo,
			config:         imagePushConfig,
			layersPushed:   pushMap{layersPushed: make(map[digest.Digest]bool)},
		}, nil
	case registry.APIVersion1:
		return &v1Pusher{
			v1IDService: metadata.NewV1IDService(imagePushConfig.MetadataStore),
			ref:         ref,
			endpoint:    endpoint,
			repoInfo:    repoInfo,
			config:      imagePushConfig,
		}, nil
	}
	return nil, fmt.Errorf("unknown version %d for registry %s", endpoint.Version, endpoint.URL)
}

// Push initiates a push operation on the repository named localName.
// ref is the specific variant of the image to be pushed.
// If no tag is provided, all tags will be pushed.
func Push(ctx context.Context, ref reference.Named, imagePushConfig *ImagePushConfig) error {
	// FIXME: Allow to interrupt current push when new push of same image is done.

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := imagePushConfig.RegistryService.ResolveRepository(ref)
	if err != nil {
		return err
	}

	endpoints, err := imagePushConfig.RegistryService.LookupPushEndpoints(repoInfo.CanonicalName)
	if err != nil {
		return err
	}

	progress.Messagef(imagePushConfig.ProgressOutput, "", "The push refers to a repository [%s]", repoInfo.CanonicalName.String())

	associations := imagePushConfig.TagStore.ReferencesByName(repoInfo.LocalName)
	if len(associations) == 0 {
		return fmt.Errorf("Repository does not exist: %s", repoInfo.LocalName)
	}

	var lastErr error
	for _, endpoint := range endpoints {
		logrus.Debugf("Trying to push %s to %s %s", repoInfo.CanonicalName, endpoint.URL, endpoint.Version)

		pusher, err := NewPusher(ref, endpoint, repoInfo, imagePushConfig)
		if err != nil {
			lastErr = err
			continue
		}
		if fallback, err := pusher.Push(ctx); err != nil {
			// Was this push cancelled? If so, don't try to fall
			// back.
			select {
			case <-ctx.Done():
				fallback = false
			default:
			}

			if fallback {
				lastErr = err
				continue
			}
			logrus.Debugf("Not continuing with error: %v", err)
			return err

		}

		imagePushConfig.EventsService.Log("push", repoInfo.LocalName.Name(), "")
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no endpoints found for %s", repoInfo.CanonicalName)
	}
	return lastErr
}

// compress returns an io.ReadCloser which will supply a compressed version of
// the provided Reader. The caller must close the ReadCloser after reading the
// compressed data.
//
// Note that this function returns a reader instead of taking a writer as an
// argument so that it can be used with httpBlobWriter's ReadFrom method.
// Using httpBlobWriter's Write method would send a PATCH request for every
// Write call.
func compress(in io.Reader) io.ReadCloser {
	pipeReader, pipeWriter := io.Pipe()
	// Use a bufio.Writer to avoid excessive chunking in HTTP request.
	bufWriter := bufio.NewWriterSize(pipeWriter, compressionBufSize)
	compressor := gzip.NewWriter(bufWriter)

	go func() {
		_, err := io.Copy(compressor, in)
		if err == nil {
			err = compressor.Close()
		}
		if err == nil {
			err = bufWriter.Flush()
		}
		if err != nil {
			pipeWriter.CloseWithError(err)
		} else {
			pipeWriter.Close()
		}
	}()

	return pipeReader
}
