package daemon

import (
	"io"
	"runtime"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/distribution"
	progressutils "github.com/tiborvass/docker/distribution/utils"
	"github.com/tiborvass/docker/pkg/progress"
	"golang.org/x/net/context"
)

// PushImage initiates a push operation on the repository named localName.
func (daemon *Daemon) PushImage(ctx context.Context, image, tag string, metaHeaders map[string][]string, authConfig *types.AuthConfig, outStream io.Writer) error {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}
	if tag != "" {
		// Push by digest is not supported, so only tags are supported.
		ref, err = reference.WithTag(ref, tag)
		if err != nil {
			return err
		}
	}

	// Include a buffer so that slow client connections don't affect
	// transfer performance.
	progressChan := make(chan progress.Progress, 100)

	writesDone := make(chan struct{})

	ctx, cancelFunc := context.WithCancel(ctx)

	go func() {
		progressutils.WriteDistributionProgress(cancelFunc, outStream, progressChan)
		close(writesDone)
	}()

	// ------------------------------------------------------------------------------
	// TODO @jhowardmsft LCOW. For now, use just the store for the host OS. This will
	// need some work to complete.
	// ------------------------------------------------------------------------------

	imagePushConfig := &distribution.ImagePushConfig{
		Config: distribution.Config{
			MetaHeaders:      metaHeaders,
			AuthConfig:       authConfig,
			ProgressOutput:   progress.ChanOutput(progressChan),
			RegistryService:  daemon.RegistryService,
			ImageEventLogger: daemon.LogImageEvent,
			MetadataStore:    daemon.stores[runtime.GOOS].distributionMetadataStore,
			ImageStore:       distribution.NewImageConfigStoreFromStore(daemon.stores[runtime.GOOS].imageStore),
			ReferenceStore:   daemon.stores[runtime.GOOS].referenceStore,
		},
		ConfigMediaType: schema2.MediaTypeImageConfig,
		LayerStore:      distribution.NewLayerProviderFromStore(daemon.stores[runtime.GOOS].layerStore),
		TrustKey:        daemon.trustKey,
		UploadManager:   daemon.uploadManager,
	}

	err = distribution.Push(ctx, ref, imagePushConfig)
	close(progressChan)
	<-writesDone
	return err
}
