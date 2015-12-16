package distribution

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/tiborvass/docker/distribution/metadata"
	"github.com/tiborvass/docker/distribution/xfer"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/image/v1"
	"github.com/tiborvass/docker/layer"
	"github.com/tiborvass/docker/pkg/ioutils"
	"github.com/tiborvass/docker/pkg/progress"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/tiborvass/docker/reference"
	"github.com/tiborvass/docker/registry"
	"golang.org/x/net/context"
)

type v2Puller struct {
	blobSumService *metadata.BlobSumService
	endpoint       registry.APIEndpoint
	config         *ImagePullConfig
	repoInfo       *registry.RepositoryInfo
	repo           distribution.Repository
}

func (p *v2Puller) Pull(ctx context.Context, ref reference.Named) (fallback bool, err error) {
	// TODO(tiborvass): was ReceiveTimeout
	p.repo, err = NewV2Repository(p.repoInfo, p.endpoint, p.config.MetaHeaders, p.config.AuthConfig, "pull")
	if err != nil {
		logrus.Warnf("Error getting v2 registry: %v", err)
		return true, err
	}

	if err := p.pullV2Repository(ctx, ref); err != nil {
		if registry.ContinueOnError(err) {
			logrus.Debugf("Error trying v2 registry: %v", err)
			return true, err
		}
		return false, err
	}
	return false, nil
}

func (p *v2Puller) pullV2Repository(ctx context.Context, ref reference.Named) (err error) {
	var refs []reference.Named
	if !reference.IsNameOnly(ref) {
		refs = []reference.Named{ref}
	} else {
		manSvc, err := p.repo.Manifests(ctx)
		if err != nil {
			return err
		}

		tags, err := manSvc.Tags()
		if err != nil {
			return err
		}

		// This probably becomes a lot nicer after the manifest
		// refactor...
		for _, tag := range tags {
			tagRef, err := reference.WithTag(ref, tag)
			if err != nil {
				return err
			}
			refs = append(refs, tagRef)
		}
	}

	var layersDownloaded bool
	for _, pullRef := range refs {
		// pulledNew is true if either new layers were downloaded OR if existing images were newly tagged
		// TODO(tiborvass): should we change the name of `layersDownload`? What about message in WriteStatus?
		pulledNew, err := p.pullV2Tag(ctx, pullRef)
		if err != nil {
			return err
		}
		layersDownloaded = layersDownloaded || pulledNew
	}

	writeStatus(ref.String(), p.config.ProgressOutput, layersDownloaded)

	return nil
}

type v2LayerDescriptor struct {
	digest         digest.Digest
	repo           distribution.Repository
	blobSumService *metadata.BlobSumService
}

func (ld *v2LayerDescriptor) Key() string {
	return "v2:" + ld.digest.String()
}

func (ld *v2LayerDescriptor) ID() string {
	return stringid.TruncateID(ld.digest.String())
}

func (ld *v2LayerDescriptor) DiffID() (layer.DiffID, error) {
	return ld.blobSumService.GetDiffID(ld.digest)
}

func (ld *v2LayerDescriptor) Download(ctx context.Context, progressOutput progress.Output) (io.ReadCloser, int64, error) {
	logrus.Debugf("pulling blob %q", ld.digest)

	blobs := ld.repo.Blobs(ctx)

	layerDownload, err := blobs.Open(ctx, ld.digest)
	if err != nil {
		logrus.Debugf("Error statting layer: %v", err)
		if err == distribution.ErrBlobUnknown {
			return nil, 0, xfer.DoNotRetry{Err: err}
		}
		return nil, 0, retryOnError(err)
	}

	size, err := layerDownload.Seek(0, os.SEEK_END)
	if err != nil {
		// Seek failed, perhaps because there was no Content-Length
		// header. This shouldn't fail the download, because we can
		// still continue without a progress bar.
		size = 0
	} else {
		// Restore the seek offset at the beginning of the stream.
		_, err = layerDownload.Seek(0, os.SEEK_SET)
		if err != nil {
			return nil, 0, err
		}
	}

	reader := progress.NewProgressReader(ioutils.NewCancelReadCloser(ctx, layerDownload), progressOutput, size, ld.ID(), "Downloading")
	defer reader.Close()

	verifier, err := digest.NewDigestVerifier(ld.digest)
	if err != nil {
		return nil, 0, xfer.DoNotRetry{Err: err}
	}

	tmpFile, err := ioutil.TempFile("", "GetImageBlob")
	if err != nil {
		return nil, 0, xfer.DoNotRetry{Err: err}
	}

	_, err = io.Copy(tmpFile, io.TeeReader(reader, verifier))
	if err != nil {
		return nil, 0, retryOnError(err)
	}

	progress.Update(progressOutput, ld.ID(), "Verifying Checksum")

	if !verifier.Verified() {
		err = fmt.Errorf("filesystem layer verification failed for digest %s", ld.digest)
		logrus.Error(err)
		tmpFile.Close()
		if err := os.RemoveAll(tmpFile.Name()); err != nil {
			logrus.Errorf("Failed to remove temp file: %s", tmpFile.Name())
		}

		return nil, 0, xfer.DoNotRetry{Err: err}
	}

	progress.Update(progressOutput, ld.ID(), "Download complete")

	logrus.Debugf("Downloaded %s to tempfile %s", ld.ID(), tmpFile.Name())

	tmpFile.Seek(0, 0)
	return ioutils.NewReadCloserWrapper(tmpFile, tmpFileCloser(tmpFile)), size, nil
}

func (ld *v2LayerDescriptor) Registered(diffID layer.DiffID) {
	// Cache mapping from this layer's DiffID to the blobsum
	ld.blobSumService.Add(diffID, ld.digest)
}

func (p *v2Puller) pullV2Tag(ctx context.Context, ref reference.Named) (tagUpdated bool, err error) {
	tagOrDigest := ""
	if tagged, isTagged := ref.(reference.NamedTagged); isTagged {
		tagOrDigest = tagged.Tag()
	} else if digested, isCanonical := ref.(reference.Canonical); isCanonical {
		tagOrDigest = digested.Digest().String()
	} else {
		return false, fmt.Errorf("internal error: reference has neither a tag nor a digest: %s", ref.String())
	}

	logrus.Debugf("Pulling ref from V2 registry: %q", tagOrDigest)

	manSvc, err := p.repo.Manifests(ctx)
	if err != nil {
		return false, err
	}

	unverifiedManifest, err := manSvc.GetByTag(tagOrDigest)
	if err != nil {
		return false, err
	}
	if unverifiedManifest == nil {
		return false, fmt.Errorf("image manifest does not exist for tag or digest %q", tagOrDigest)
	}
	var verifiedManifest *schema1.Manifest
	verifiedManifest, err = verifyManifest(unverifiedManifest, ref)
	if err != nil {
		return false, err
	}

	rootFS := image.NewRootFS()

	if err := detectBaseLayer(p.config.ImageStore, verifiedManifest, rootFS); err != nil {
		return false, err
	}

	// remove duplicate layers and check parent chain validity
	err = fixManifestLayers(verifiedManifest)
	if err != nil {
		return false, err
	}

	progress.Message(p.config.ProgressOutput, tagOrDigest, "Pulling from "+p.repo.Name())

	var descriptors []xfer.DownloadDescriptor

	// Image history converted to the new format
	var history []image.History

	// Note that the order of this loop is in the direction of bottom-most
	// to top-most, so that the downloads slice gets ordered correctly.
	for i := len(verifiedManifest.FSLayers) - 1; i >= 0; i-- {
		blobSum := verifiedManifest.FSLayers[i].BlobSum

		var throwAway struct {
			ThrowAway bool `json:"throwaway,omitempty"`
		}
		if err := json.Unmarshal([]byte(verifiedManifest.History[i].V1Compatibility), &throwAway); err != nil {
			return false, err
		}

		h, err := v1.HistoryFromConfig([]byte(verifiedManifest.History[i].V1Compatibility), throwAway.ThrowAway)
		if err != nil {
			return false, err
		}
		history = append(history, h)

		if throwAway.ThrowAway {
			continue
		}

		layerDescriptor := &v2LayerDescriptor{
			digest:         blobSum,
			repo:           p.repo,
			blobSumService: p.blobSumService,
		}

		descriptors = append(descriptors, layerDescriptor)
	}

	resultRootFS, release, err := p.config.DownloadManager.Download(ctx, *rootFS, descriptors, p.config.ProgressOutput)
	if err != nil {
		return false, err
	}
	defer release()

	config, err := v1.MakeConfigFromV1Config([]byte(verifiedManifest.History[0].V1Compatibility), &resultRootFS, history)
	if err != nil {
		return false, err
	}

	imageID, err := p.config.ImageStore.Create(config)
	if err != nil {
		return false, err
	}

	manifestDigest, _, err := digestFromManifest(unverifiedManifest, p.repoInfo)
	if err != nil {
		return false, err
	}

	if manifestDigest != "" {
		progress.Message(p.config.ProgressOutput, "", "Digest: "+manifestDigest.String())
	}

	oldTagImageID, err := p.config.ReferenceStore.Get(ref)
	if err == nil && oldTagImageID == imageID {
		return false, nil
	}

	if canonical, ok := ref.(reference.Canonical); ok {
		if err = p.config.ReferenceStore.AddDigest(canonical, imageID, true); err != nil {
			return false, err
		}
	} else if err = p.config.ReferenceStore.AddTag(ref, imageID, true); err != nil {
		return false, err
	}

	return true, nil
}

func verifyManifest(signedManifest *schema1.SignedManifest, ref reference.Named) (m *schema1.Manifest, err error) {
	// If pull by digest, then verify the manifest digest. NOTE: It is
	// important to do this first, before any other content validation. If the
	// digest cannot be verified, don't even bother with those other things.
	if digested, isCanonical := ref.(reference.Canonical); isCanonical {
		verifier, err := digest.NewDigestVerifier(digested.Digest())
		if err != nil {
			return nil, err
		}
		payload, err := signedManifest.Payload()
		if err != nil {
			// If this failed, the signatures section was corrupted
			// or missing. Treat the entire manifest as the payload.
			payload = signedManifest.Raw
		}
		if _, err := verifier.Write(payload); err != nil {
			return nil, err
		}
		if !verifier.Verified() {
			err := fmt.Errorf("image verification failed for digest %s", digested.Digest())
			logrus.Error(err)
			return nil, err
		}

		var verifiedManifest schema1.Manifest
		if err = json.Unmarshal(payload, &verifiedManifest); err != nil {
			return nil, err
		}
		m = &verifiedManifest
	} else {
		m = &signedManifest.Manifest
	}

	if m.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema version %d for %q", m.SchemaVersion, ref.String())
	}
	if len(m.FSLayers) != len(m.History) {
		return nil, fmt.Errorf("length of history not equal to number of layers for %q", ref.String())
	}
	if len(m.FSLayers) == 0 {
		return nil, fmt.Errorf("no FSLayers in manifest for %q", ref.String())
	}
	return m, nil
}

// fixManifestLayers removes repeated layers from the manifest and checks the
// correctness of the parent chain.
func fixManifestLayers(m *schema1.Manifest) error {
	imgs := make([]*image.V1Image, len(m.FSLayers))
	for i := range m.FSLayers {
		img := &image.V1Image{}

		if err := json.Unmarshal([]byte(m.History[i].V1Compatibility), img); err != nil {
			return err
		}

		imgs[i] = img
		if err := v1.ValidateID(img.ID); err != nil {
			return err
		}
	}

	if imgs[len(imgs)-1].Parent != "" && runtime.GOOS != "windows" {
		// Windows base layer can point to a base layer parent that is not in manifest.
		return errors.New("Invalid parent ID in the base layer of the image.")
	}

	// check general duplicates to error instead of a deadlock
	idmap := make(map[string]struct{})

	var lastID string
	for _, img := range imgs {
		// skip IDs that appear after each other, we handle those later
		if _, exists := idmap[img.ID]; img.ID != lastID && exists {
			return fmt.Errorf("ID %+v appears multiple times in manifest", img.ID)
		}
		lastID = img.ID
		idmap[lastID] = struct{}{}
	}

	// backwards loop so that we keep the remaining indexes after removing items
	for i := len(imgs) - 2; i >= 0; i-- {
		if imgs[i].ID == imgs[i+1].ID { // repeated ID. remove and continue
			m.FSLayers = append(m.FSLayers[:i], m.FSLayers[i+1:]...)
			m.History = append(m.History[:i], m.History[i+1:]...)
		} else if imgs[i].Parent != imgs[i+1].ID {
			return fmt.Errorf("Invalid parent ID. Expected %v, got %v.", imgs[i+1].ID, imgs[i].Parent)
		}
	}

	return nil
}
