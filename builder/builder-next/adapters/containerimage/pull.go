package containerimage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	ctdlabels "github.com/containerd/containerd/labels"
	"github.com/containerd/containerd/leases"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/snapshots"
	ctdreference "github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/remotes/docker/schema1"
	distreference "github.com/docker/distribution/reference"
	"github.com/docker/docker/distribution"
	"github.com/docker/docker/distribution/metadata"
	"github.com/docker/docker/distribution/xfer"
	"github.com/docker/docker/image"
	"github.com/docker/docker/layer"
	pkgprogress "github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/reference"
	"github.com/moby/buildkit/cache"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/solver"
	"github.com/moby/buildkit/source"
	"github.com/moby/buildkit/util/contentutil"
	"github.com/moby/buildkit/util/flightcontrol"
	"github.com/moby/buildkit/util/imageutil"
	"github.com/moby/buildkit/util/leaseutil"
	"github.com/moby/buildkit/util/progress"
	"github.com/moby/buildkit/util/progress/controller"
	"github.com/moby/buildkit/util/resolver"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// SourceOpt is options for creating the image source
type SourceOpt struct {
	ContentStore    content.Store
	CacheAccessor   cache.Accessor
	ReferenceStore  reference.Store
	DownloadManager distribution.RootFSDownloadManager
	MetadataStore   metadata.V2MetadataService
	ImageStore      image.Store
	RegistryHosts   docker.RegistryHosts
	LayerStore      layer.Store
	LeaseManager    leases.Manager
}

// Source is the source implementation for accessing container images
type Source struct {
	SourceOpt
	g             flightcontrol.Group
}

// NewSource creates a new image source
func NewSource(opt SourceOpt) (*Source, error) {
	return &Source{SourceOpt: opt}, nil
}

// ID returns image scheme identifier
func (is *Source) ID() string {
	return source.DockerImageScheme
}

func (is *Source) resolveLocal(refStr string) (*image.Image, error) {
	ref, err := distreference.ParseNormalizedNamed(refStr)
	if err != nil {
		return nil, err
	}
	dgst, err := is.ReferenceStore.Get(ref)
	if err != nil {
		return nil, err
	}
	img, err := is.ImageStore.Get(image.ID(dgst))
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (is *Source) resolveRemote(ctx context.Context, ref string, platform *ocispec.Platform, sm *session.Manager, g session.Group) (digest.Digest, []byte, error) {
	type t struct {
		dgst digest.Digest
		dt   []byte
	}
	res, err := is.g.Do(ctx, ref, func(ctx context.Context) (interface{}, error) {
		res := resolver.DefaultPool.GetResolver(is.RegistryHosts, ref, "pull", sm, g)
		dgst, dt, err := imageutil.Config(ctx, ref, res, is.ContentStore, nil, platform)
		if err != nil {
			return nil, err
		}
		return &t{dgst: dgst, dt: dt}, nil
	})
	var typed *t
	if err != nil {
		return "", nil, err
	}
	typed = res.(*t)
	return typed.dgst, typed.dt, nil
}

// ResolveImageConfig returns image config for an image
func (is *Source) ResolveImageConfig(ctx context.Context, ref string, opt llb.ResolveImageConfigOpt, sm *session.Manager, g session.Group) (digest.Digest, []byte, error) {
	resolveMode, err := source.ParseImageResolveMode(opt.ResolveMode)
	if err != nil {
		return "", nil, err
	}
	switch resolveMode {
	case source.ResolveModeForcePull:
		dgst, dt, err := is.resolveRemote(ctx, ref, opt.Platform, sm, g)
		// TODO: pull should fallback to local in case of failure to allow offline behavior
		// the fallback doesn't work currently
		return dgst, dt, err
		/*
			if err == nil {
				return dgst, dt, err
			}
			// fallback to local
			dt, err = is.resolveLocal(ref)
			return "", dt, err
		*/

	case source.ResolveModeDefault:
		// default == prefer local, but in the future could be smarter
		fallthrough
	case source.ResolveModePreferLocal:
		img, err := is.resolveLocal(ref)
		if err == nil {
			if opt.Platform != nil && !platformMatches(img, opt.Platform) {
				logrus.WithField("ref", ref).Debugf("Requested build platform %s does not match local image platform %s, checking remote",
					path.Join(opt.Platform.OS, opt.Platform.Architecture, opt.Platform.Variant),
					path.Join(img.OS, img.Architecture, img.Variant),
				)
			} else {
				return "", img.RawJSON(), err
			}
		}
		// fallback to remote
		return is.resolveRemote(ctx, ref, opt.Platform, sm, g)
	}
	// should never happen
	return "", nil, fmt.Errorf("builder cannot resolve image %s: invalid mode %q", ref, opt.ResolveMode)
}

// Resolve returns access to pulling for an identifier
func (is *Source) Resolve(ctx context.Context, id source.Identifier, sm *session.Manager, vtx solver.Vertex) (source.SourceInstance, error) {
	imageIdentifier, ok := id.(*source.ImageIdentifier)
	if !ok {
		return nil, errors.Errorf("invalid image identifier %v", id)
	}

	platform := platforms.DefaultSpec()
	if imageIdentifier.Platform != nil {
		platform = *imageIdentifier.Platform
	}

	p := &puller{
		src: imageIdentifier,
		is:  is,
		platform: platform,
		sm:       sm,
		vtx: vtx,
	}
	return p, nil
}

type puller struct {
	is               *Source
	ref              string
	sm               *session.Manager
	src              *source.ImageIdentifier
	vtx              solver.Vertex

	cacheKeyOnce     sync.Once
	cacheKeyErr      error
	releaseTmpLeases func(context.Context) error
	descHandlers     cache.DescHandlers
	//manifest         *pull.PulledManifests
	manifestRef      string
	manifestKey      string
	configKey        string

	// Equivalent of github.com/moby/buildkit/util/pull.Puller

	resolverOnce     sync.Once
	resolverInstance remotes.Resolver
	platform         ocispec.Platform
	resolveOnce      sync.Once
	resolveLocalOnce sync.Once
	resolveErr       error
	desc             ocispec.Descriptor
	configDesc       ocispec.Descriptor
	config           []byte
	layers           []ocispec.Descriptor
}

func (p *puller) resolver(g session.Group) remotes.Resolver {
	p.resolverOnce.Do(func() {
		if p.resolverInstance == nil {
			p.resolverInstance = resolver.DefaultPool.GetResolver(p.is.RegistryHosts, p.src.Reference.String(), "pull", p.sm, g)
		}
	})
	return p.resolverInstance
}

func mainManifestKey(dgst digest.Digest, platform ocispec.Platform) (digest.Digest, error) {
	dt, err := json.Marshal(struct {
		Digest  digest.Digest
		OS      string
		Arch    string
		Variant string `json:",omitempty"`
	}{
		Digest:  dgst,
		OS:      platform.OS,
		Arch:    platform.Architecture,
		Variant: platform.Variant,
	})
	if err != nil {
		return "", err
	}
	return digest.FromBytes(dt), nil
}

func (p *puller) resolveLocal() {
	p.resolveLocalOnce.Do(func() {
		dgst := p.src.Reference.Digest()
		if dgst != "" {
			info, err := p.is.ContentStore.Info(context.TODO(), dgst)
			if err == nil {
				p.ref = p.src.Reference.String()
				desc := ocispec.Descriptor{
					Size:   info.Size,
					Digest: dgst,
				}
				ra, err := p.is.ContentStore.ReaderAt(context.TODO(), desc)
				if err == nil {
					mt, err := imageutil.DetectManifestMediaType(ra)
					if err == nil {
						desc.MediaType = mt
						p.desc = desc
					}
				}
			}
		}

		if p.src.ResolveMode == source.ResolveModeDefault || p.src.ResolveMode == source.ResolveModePreferLocal {
			ref := p.src.Reference.String()
			img, err := p.is.resolveLocal(ref)
			if err == nil {
				if !platformMatches(img, &p.platform) {
					logrus.WithField("ref", ref).Debugf("Requested build platform %s does not match local image platform %s, not resolving",
						path.Join(p.platform.OS, p.platform.Architecture, p.platform.Variant),
						path.Join(img.OS, img.Architecture, img.Variant),
					)
				} else {
					p.config = img.RawJSON()
				}
			}
		}
	})
}

func (p *puller) resolve(ctx context.Context, g session.Group /* TODO: remove and rely on resolver(g) instead */) (err error) {
	p.resolveOnce.Do(func() {
		// TODO: is this really needed here? Can't we use p.src.Reference.String() in Resolve() below?
		ref, err := distreference.ParseNormalizedNamed(p.src.Reference.String())
		if err != nil {
			p.resolveErr = err
			return
		}

		if p.desc.Digest == "" && p.config == nil {
			origRef, desc, err := p.resolverInstance.Resolve(ctx, ref.String())
			if err != nil {
				p.resolveErr = err
				return
			}

			p.desc = desc
			p.ref = origRef
		}

		// Schema 1 manifests cannot be resolved to an image config
		// since the conversion must take place after all the content
		// has been read.
		// It may be possible to have a mapping between schema 1 manifests
		// and the schema 2 manifests they are converted to.
		if p.config == nil && p.desc.MediaType != images.MediaTypeDockerSchema1Manifest {
			ref, err := distreference.WithDigest(ref, p.desc.Digest)
			if err != nil {
				p.resolveErr = err
				return
			}
			_, dt, err := p.is.ResolveImageConfig(ctx, ref.String(), llb.ResolveImageConfigOpt{Platform: &p.platform, ResolveMode: resolveModeToString(p.src.ResolveMode)}, p.sm, g)
			if err != nil {
				p.resolveErr = err
				return
			}

			p.config = dt
		}
	})
	return p.resolveErr
}

func (p *puller) CacheKey(ctx context.Context, g session.Group, index int) (string, solver.CacheOpts, bool, error) {
	// initialize resolver instance
	p.resolver(g)

	p.cacheKeyOnce.Do(func() {
		ctx, done, err := leaseutil.WithLease(ctx, p.is.LeaseManager, leases.WithExpiration(5*time.Minute), leaseutil.MakeTemporary)
		if err != nil {
			p.cacheKeyErr = err
			return
		}
		p.releaseTmpLeases = done
		imageutil.AddLease(p.releaseTmpLeases)
		defer func() {
			if p.cacheKeyErr != nil {
				p.releaseTmpLeases(ctx)
			}
		}()

		resolveProgressDone := oneOffProgress(ctx, "resolve "+p.src.Reference.String())
		defer func() {
			resolveProgressDone(err)
		}()

		// Follows the equivalent of github.com/moby/buildkit/util/pull.Puller.PullManifests

		p.resolveLocal()
		if err := p.resolve(ctx, g); err != nil {
			p.cacheKeyErr = err
			return
		}

		platform := platforms.Only(p.platform)

		var mu sync.Mutex // images.Dispatch calls handlers in parallel
		metadata := make(map[digest.Digest]ocispec.Descriptor)

		var (
			schema1Converter *schema1.Converter
			handlers         []images.Handler
		)

		fetcher, err := p.resolver(g).Fetcher(ctx, p.ref)
		if err != nil {
			p.cacheKeyErr = err
			return
		}

		if p.desc.MediaType == images.MediaTypeDockerSchema1Manifest {
			// TODO: fetcher was &pullprogress.FetcherWithProgress{...} in buildkit
			schema1Converter = schema1.NewConverter(p.is.ContentStore, fetcher)
			handlers = append(handlers, schema1Converter)

			// TODO: Optimize to do dispatch and integrate pulling with download manager,
			// leverage existing blob mapping and layer storage
		} else {

			// TODO: need a wrapper snapshot interface that combines content
			// and snapshots as 1) buildkit shouldn't have a dependency on contentstore
			// or 2) cachemanager should manage the contentstore
			handlers = append(handlers, images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
				switch desc.MediaType {
				case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest,
					images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex,
					images.MediaTypeDockerSchema2Config, ocispec.MediaTypeImageConfig:
				default:
					return nil, images.ErrSkipDesc
				}
				//ongoing.add(desc)
				return nil, nil
			}))

			// Get all the children for a descriptor
			childrenHandler := images.ChildrenHandler(p.is.ContentStore)
			// Set any children labels for that content
			childrenHandler = images.SetChildrenLabels(p.is.ContentStore, childrenHandler)
			// Filter the children by the platform
			childrenHandler = images.FilterPlatforms(childrenHandler, platform)
			// Limit manifests pulled to the best match in an index
			childrenHandler = images.LimitManifests(childrenHandler, platform, 1)

			handlers = append(handlers,
				filterLayerBlobs(metadata, &mu),
				remotes.FetchHandler(p.is.ContentStore, fetcher),
				childrenHandler,
			)
		}

		if err := images.Dispatch(ctx, images.Handlers(handlers...), nil, p.desc); err != nil {
			p.cacheKeyErr = err
			return
		}

		if schema1Converter != nil {
			p.desc, err = schema1Converter.Convert(ctx)
			if err != nil {
				p.cacheKeyErr = err
				return
			}
			// TODO: images.Dispatch
		}

		for _, desc := range metadata {
			// TODO: nonlayers
			switch desc.MediaType {
			case images.MediaTypeDockerSchema2Config, ocispec.MediaTypeImageConfig:
				p.configDesc = desc
			}
		}

		p.layers, p.cacheKeyErr = getLayers(ctx, p.is.ContentStore, p.desc, platform)
		if p.cacheKeyErr != nil {
			return
		}

		// end of PullManifests

		if len(p.layers) > 0 {
			pw, _, _ := progress.FromContext(ctx)
			progressController := &controller.Controller{
				Writer: pw,
			}
			if p.vtx != nil {
				progressController.Digest = p.vtx.Digest()
				progressController.Name = p.vtx.Name()
			}

			p.descHandlers = cache.DescHandlers(make(map[digest.Digest]*cache.DescHandler))
			for i, desc := range p.layers {

				// Hints for remote/stargz snapshotter for searching for remote snapshots
				labels := snapshots.FilterInheritedLabels(desc.Annotations)
				if labels == nil {
					labels = make(map[string]string)
				}
				labels["containerd.io/snapshot/remote/stargz.reference"] = p.ref
				labels["containerd.io/snapshot/remote/stargz.digest"] = desc.Digest.String()
				var (
					layersKey = "containerd.io/snapshot/remote/stargz.layers"
					layers    string
				)
				for _, l := range p.layers[i:]{
					ls := fmt.Sprintf("%s,", l.Digest.String())
					// This avoids the label hits the size limitation.
					// Skipping layers is allowed here and only affects performance.
					if err := ctdlabels.Validate(layersKey, layers+ls); err != nil {
						break
					}
					layers += ls
				}
				labels[layersKey] = strings.TrimSuffix(layers, ",")

				p.descHandlers[desc.Digest] = &cache.DescHandler{
					Provider:       p,
					Progress:       progressController,
					SnapshotLabels: labels,
				}
			}
		}
		p.manifestRef = p.ref
		k, err := mainManifestKey(p.desc.Digest, p.platform)
		if err != nil {
			p.cacheKeyErr = err
			return
		}
		p.manifestKey = k.String()

		dt, err := content.ReadBlob(ctx, p.is.ContentStore, p.configDesc)
		if err != nil {
			p.cacheKeyErr = err
			return
		}
		p.configKey = cacheKeyFromConfig(dt).String()

	})
	if p.cacheKeyErr != nil {
		return "", nil, false, p.cacheKeyErr
	}

	cacheOpts := solver.CacheOpts(make(map[interface{}]interface{}))
	for dgst, descHandler := range p.descHandlers {
		cacheOpts[cache.DescHandlerKey(dgst)] = descHandler
	}

	cacheDone := index > 0
	if index == 0 || p.configKey == "" {
		return p.manifestKey, cacheOpts, cacheDone, nil
	}
	return p.configKey, cacheOpts, cacheDone, nil
}

func (p *puller) getRef(ctx context.Context, diffIDs []layer.DiffID, opts ...cache.RefOption) (cache.ImmutableRef, error) {
	var parent cache.ImmutableRef
	if len(diffIDs) > 1 {
		var err error
		parent, err = p.getRef(ctx, diffIDs[:len(diffIDs)-1], opts...)
		if err != nil {
			return nil, err
		}
		defer parent.Release(context.TODO())
	}
	return p.is.CacheAccessor.GetByBlob(ctx, ocispec.Descriptor{
		Annotations: map[string]string{
			"containerd.io/uncompressed": diffIDs[len(diffIDs)-1].String(),
		},
	}, parent, opts...)
}

func (p *puller) Snapshot(ctx context.Context, g session.Group) (ir cache.ImmutableRef, err error) {
	p.resolver(g)
	p.resolveLocal()

	pw, _, ctx := progress.FromContext(ctx)
	defer pw.Close()

	/*
	progressDone := make(chan struct{})
	go func() {
		showProgress(pctx, ongoing, p.is.ContentStore, pw)
		close(progressDone)
	}()
	defer func() {
		<-progressDone
	}()
	*/

	pchan := make(chan pkgprogress.Progress, 10)
	defer close(pchan)

	go func() {
		m := map[string]struct {
			st      time.Time
			limiter *rate.Limiter
		}{}
		for p := range pchan {
			if p.Action == "Extracting" {
				st, ok := m[p.ID]
				if !ok {
					st.st = time.Now()
					st.limiter = rate.NewLimiter(rate.Every(100*time.Millisecond), 1)
					m[p.ID] = st
				}
				var end *time.Time
				if p.LastUpdate || st.limiter.Allow() {
					if p.LastUpdate {
						tm := time.Now()
						end = &tm
					}
					_ = pw.Write("extracting "+p.ID, progress.Status{
						Action:    "extract",
						Started:   &st.st,
						Completed: end,
					})
				}
			}
		}
	}()


	if len(p.layers) == 0 {
		return nil, nil
	}
	// TODO: do i still need this?
	defer p.releaseTmpLeases(ctx)

	if p.config != nil {
		img, err := p.is.ImageStore.Get(image.ID(digest.FromBytes(p.config)))
		if err == nil {
			if len(img.RootFS.DiffIDs) == 0 {
				return nil, nil
			}
			l, err := p.is.LayerStore.Get(img.RootFS.ChainID())
			if err == nil {
				layer.ReleaseAndLog(p.is.LayerStore, l)
				ref, err := p.getRef(ctx, img.RootFS.DiffIDs, cache.WithDescription(fmt.Sprintf("from local %s", p.ref)))
				if err != nil {
					return nil, err
				}
				return ref, nil
			}
		}
	}

	fetcher, err := p.resolver(g).Fetcher(ctx, p.ref)
	if err != nil {
		p.cacheKeyErr = err
		return
	}

	layers := make([]xfer.DownloadDescriptor, 0, len(p.layers))

	for _, layerDesc := range p.layers {
		// hack to get diffID
		s, ok := layerDesc.Annotations["containerd.io/uncompressed"]
		if !ok {
			panic("yolo")
		}
		diffID := digest.FromString(s)

		layers = append(layers, &layerDescriptor{
			desc: layerDesc,
			diffID:  layer.DiffID(diffID),
			fetcher: fetcher,
			ref: p.src.Reference,
			is: p.is,
		})
	}

	// TODO somehow figure out when progress is done
	/*
	defer func() {
		<-progressDone
		for _, desc := range p.layers {
			p.is.ContentStore.Delete(context.TODO(), desc.Digest)
		}
	}()
	*/

	r := image.NewRootFS()
	rootFS, release, err := p.is.DownloadManager.Download(ctx, *r, runtime.GOOS, layers, pkgprogress.ChanOutput(pchan))
	//TODO: stopProgress()?
	if err != nil {
		return nil, err
	}

	ref, err := p.getRef(ctx, rootFS.DiffIDs, cache.WithDescription(fmt.Sprintf("pulled from %s", p.ref)), p.descHandlers)
	release()
	if err != nil {
		return nil, err
	}

	// TODO: markRefLayerTypeWindows?

	if p.src.RecordType != "" && cache.GetRecordType(ref) == "" {
		if err := cache.SetRecordType(ref, p.src.RecordType); err != nil {
			return nil, err
		}
	}

	return ref, nil
}

func (p *puller) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	if err := p.resolve(ctx, nil); err != nil {
		return nil, err
	}
	fetcher, err := p.resolverInstance.Fetcher(ctx, p.ref)
	if err != nil {
		return nil, err
	}
	return contentutil.FromFetcher(fetcher).ReaderAt(ctx, desc)
}

// Fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error)
type layerDescriptor struct {
	is      *Source
	fetcher remotes.Fetcher
	desc    ocispec.Descriptor
	diffID  layer.DiffID
	ref     ctdreference.Spec
}

func (ld *layerDescriptor) Key() string {
	return "v2:" + ld.desc.Digest.String()
}

func (ld *layerDescriptor) ID() string {
	return ld.desc.Digest.String()
}

func (ld *layerDescriptor) DiffID() (layer.DiffID, error) {
	return ld.diffID, nil
}

func (ld *layerDescriptor) Download(ctx context.Context, progressOutput pkgprogress.Output) (io.ReadCloser, int64, error) {
	rc, err := ld.fetcher.Fetch(ctx, ld.desc)
	if err != nil {
		return nil, 0, err
	}
	defer rc.Close()

	refKey := remotes.MakeRefKey(ctx, ld.desc)

	ld.is.ContentStore.Abort(ctx, refKey)

	if err := content.WriteBlob(ctx, ld.is.ContentStore, refKey, rc, ld.desc); err != nil {
		ld.is.ContentStore.Abort(ctx, refKey)
		return nil, 0, err
	}

	ra, err := ld.is.ContentStore.ReaderAt(ctx, ld.desc)
	if err != nil {
		return nil, 0, err
	}

	return ioutil.NopCloser(content.NewReader(ra)), ld.desc.Size, nil
}

func (ld *layerDescriptor) Close() {
	// ld.is.ContentStore.Delete(context.TODO(), ld.desc.Digest))
}

func (ld *layerDescriptor) Registered(diffID layer.DiffID) {
	// Cache mapping from this layer's DiffID to the blobsum
	ld.is.MetadataStore.Add(diffID, metadata.V2Metadata{Digest: ld.desc.Digest, SourceRepository: ld.ref.Locator})
}

// cacheKeyFromConfig returns a stable digest from image config. If image config
// is a known oci image we will use chainID of layers.
func cacheKeyFromConfig(dt []byte) digest.Digest {
	var img ocispec.Image
	err := json.Unmarshal(dt, &img)
	if err != nil {
		return digest.FromBytes(dt)
	}
	if img.RootFS.Type != "layers" || len(img.RootFS.DiffIDs) == 0 {
		return ""
	}
	return identity.ChainID(img.RootFS.DiffIDs)
}

// resolveModeToString is the equivalent of github.com/moby/buildkit/solver/llb.ResolveMode.String()
// FIXME: add String method on source.ResolveMode
func resolveModeToString(rm source.ResolveMode) string {
	switch rm {
	case source.ResolveModeDefault:
		return "default"
	case source.ResolveModeForcePull:
		return "pull"
	case source.ResolveModePreferLocal:
		return "local"
	}
	return ""
}

func platformMatches(img *image.Image, p *ocispec.Platform) bool {
	if img.Architecture != p.Architecture {
		return false
	}
	if img.Variant != "" && img.Variant != p.Variant {
		return false
	}
	return img.OS == p.OS
}

// filterLayerBlobs causes layer blobs to be skipped for fetch, which is required to support lazy blobs.
// It also stores the non-layer blobs (metadata) it encounters in the provided map.
func filterLayerBlobs(metadata map[digest.Digest]ocispec.Descriptor, mu sync.Locker) images.HandlerFunc {
       return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
               switch desc.MediaType {
               case ocispec.MediaTypeImageLayer, images.MediaTypeDockerSchema2Layer, ocispec.MediaTypeImageLayerGzip, images.MediaTypeDockerSchema2LayerGzip, images.MediaTypeDockerSchema2LayerForeign, images.MediaTypeDockerSchema2LayerForeignGzip:
                       return nil, images.ErrSkipDesc
               default:
                       if metadata != nil {
                               mu.Lock()
                               metadata[desc.Digest] = desc
                               mu.Unlock()
                       }
               }
               return nil, nil
       }
}

func oneOffProgress(ctx context.Context, id string) func(err error) error {
	pw, _, _ := progress.FromContext(ctx)
	now := time.Now()
	st := progress.Status{
		Started: &now,
	}
	pw.Write(id, st)
	return func(err error) error {
		// TODO: set error on status
		now := time.Now()
		st.Completed = &now
		pw.Write(id, st)
		pw.Close()
		return err
	}
}

func getLayers(ctx context.Context, provider content.Provider, desc ocispec.Descriptor, platform platforms.MatchComparer) ([]ocispec.Descriptor, error) {
	manifest, err := images.Manifest(ctx, provider, desc, platform)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	image := images.Image{Target: desc}
	diffIDs, err := image.RootFS(ctx, provider, platform)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve rootfs")
	}
	if len(diffIDs) != len(manifest.Layers) {
		return nil, errors.Errorf("mismatched image rootfs and manifest layers %+v %+v", diffIDs, manifest.Layers)
	}
	layers := make([]ocispec.Descriptor, len(diffIDs))
	for i := range diffIDs {
		desc := manifest.Layers[i]
		if desc.Annotations == nil {
			desc.Annotations = map[string]string{}
		}
		desc.Annotations["containerd.io/uncompressed"] = diffIDs[i].String()
		layers[i] = desc
	}
	return layers, nil
}
