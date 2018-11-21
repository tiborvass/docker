package containerimage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"sync"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	ctdreference "github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/remotes/docker/schema1"
	distreference "github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/distribution"
	"github.com/tiborvass/docker/distribution/metadata"
	"github.com/tiborvass/docker/distribution/xfer"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/layer"
	pkgprogress "github.com/tiborvass/docker/pkg/progress"
	"github.com/tiborvass/docker/reference"
	"github.com/moby/buildkit/cache"
	gw "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"github.com/moby/buildkit/source"
	"github.com/moby/buildkit/util/flightcontrol"
	"github.com/moby/buildkit/util/imageutil"
	"github.com/moby/buildkit/util/progress"
	"github.com/moby/buildkit/util/resolver"
	"github.com/moby/buildkit/util/tracing"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// SourceOpt is options for creating the image source
type SourceOpt struct {
	SessionManager  *session.Manager
	ContentStore    content.Store
	CacheAccessor   cache.Accessor
	ReferenceStore  reference.Store
	DownloadManager distribution.RootFSDownloadManager
	MetadataStore   metadata.V2MetadataService
	ImageStore      image.Store
	ResolverOpt     resolver.ResolveOptionsFunc
}

type imageSource struct {
	SourceOpt
	g flightcontrol.Group
}

// NewSource creates a new image source
func NewSource(opt SourceOpt) (source.Source, error) {
	is := &imageSource{
		SourceOpt: opt,
	}

	return is, nil
}

func (is *imageSource) ID() string {
	return source.DockerImageScheme
}

func (is *imageSource) getResolver(ctx context.Context, rfn resolver.ResolveOptionsFunc, ref string) remotes.Resolver {
	opt := docker.ResolverOptions{
		Client: tracing.DefaultClient,
	}
	if rfn != nil {
		opt = rfn(ref)
	}
	opt.Credentials = is.getCredentialsFromSession(ctx)
	r := docker.NewResolver(opt)
	return r
}

func (is *imageSource) getCredentialsFromSession(ctx context.Context) func(string) (string, string, error) {
	id := session.FromContext(ctx)
	if id == "" {
		// can be removed after containerd/containerd#2812
		return func(string) (string, string, error) {
			return "", "", nil
		}
	}
	return func(host string) (string, string, error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		caller, err := is.SessionManager.Get(timeoutCtx, id)
		if err != nil {
			return "", "", err
		}

		return auth.CredentialsFunc(tracing.ContextWithSpanFromContext(context.TODO(), ctx), caller)(host)
	}
}

func (is *imageSource) resolveLocal(refStr string) ([]byte, error) {
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
	return img.RawJSON(), nil
}

func (is *imageSource) resolveRemote(ctx context.Context, ref string, platform *ocispec.Platform) (digest.Digest, []byte, error) {
	type t struct {
		dgst digest.Digest
		dt   []byte
	}
	res, err := is.g.Do(ctx, ref, func(ctx context.Context) (interface{}, error) {
		dgst, dt, err := imageutil.Config(ctx, ref, is.getResolver(ctx, is.ResolverOpt, ref), is.ContentStore, platform)
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

func (is *imageSource) ResolveImageConfig(ctx context.Context, ref string, opt gw.ResolveImageConfigOpt) (digest.Digest, []byte, error) {
	resolveMode, err := source.ParseImageResolveMode(opt.ResolveMode)
	if err != nil {
		return "", nil, err
	}
	switch resolveMode {
	case source.ResolveModeForcePull:
		dgst, dt, err := is.resolveRemote(ctx, ref, opt.Platform)
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
		dt, err := is.resolveLocal(ref)
		if err == nil {
			return "", dt, err
		}
		// fallback to remote
		return is.resolveRemote(ctx, ref, opt.Platform)
	}
	// should never happen
	return "", nil, fmt.Errorf("builder cannot resolve image %s: invalid mode %q", ref, opt.ResolveMode)
}

func (is *imageSource) Resolve(ctx context.Context, id source.Identifier) (source.SourceInstance, error) {
	imageIdentifier, ok := id.(*source.ImageIdentifier)
	if !ok {
		return nil, errors.Errorf("invalid image identifier %v", id)
	}

	platform := platforms.DefaultSpec()
	if imageIdentifier.Platform != nil {
		platform = *imageIdentifier.Platform
	}

	p := &puller{
		src:      imageIdentifier,
		is:       is,
		resolver: is.getResolver(ctx, is.ResolverOpt, imageIdentifier.Reference.String()),
		platform: platform,
	}
	return p, nil
}

type puller struct {
	is               *imageSource
	resolveOnce      sync.Once
	resolveLocalOnce sync.Once
	src              *source.ImageIdentifier
	desc             ocispec.Descriptor
	ref              string
	resolveErr       error
	resolver         remotes.Resolver
	config           []byte
	platform         ocispec.Platform
}

func (p *puller) mainManifestKey(dgst digest.Digest, platform ocispec.Platform) (digest.Digest, error) {
	dt, err := json.Marshal(struct {
		Digest  digest.Digest
		OS      string
		Arch    string
		Variant string `json:",omitempty"`
	}{
		Digest:  p.desc.Digest,
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
			dt, err := p.is.resolveLocal(p.src.Reference.String())
			if err == nil {
				p.config = dt
			}
		}
	})
}

func (p *puller) resolve(ctx context.Context) error {
	p.resolveOnce.Do(func() {
		resolveProgressDone := oneOffProgress(ctx, "resolve "+p.src.Reference.String())

		ref, err := distreference.ParseNormalizedNamed(p.src.Reference.String())
		if err != nil {
			p.resolveErr = err
			resolveProgressDone(err)
			return
		}

		if p.desc.Digest == "" && p.config == nil {
			origRef, desc, err := p.resolver.Resolve(ctx, ref.String())
			if err != nil {
				p.resolveErr = err
				resolveProgressDone(err)
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
				resolveProgressDone(err)
				return
			}
			_, dt, err := p.is.ResolveImageConfig(ctx, ref.String(), gw.ResolveImageConfigOpt{Platform: &p.platform, ResolveMode: resolveModeToString(p.src.ResolveMode)})
			if err != nil {
				p.resolveErr = err
				resolveProgressDone(err)
				return
			}

			p.config = dt
		}
		resolveProgressDone(nil)
	})
	return p.resolveErr
}

func (p *puller) CacheKey(ctx context.Context, index int) (string, bool, error) {
	p.resolveLocal()

	if p.desc.Digest != "" && index == 0 {
		dgst, err := p.mainManifestKey(p.desc.Digest, p.platform)
		if err != nil {
			return "", false, err
		}
		return dgst.String(), false, nil
	}

	if p.config != nil {
		return cacheKeyFromConfig(p.config).String(), true, nil
	}

	if err := p.resolve(ctx); err != nil {
		return "", false, err
	}

	if p.desc.Digest != "" && index == 0 {
		dgst, err := p.mainManifestKey(p.desc.Digest, p.platform)
		if err != nil {
			return "", false, err
		}
		return dgst.String(), false, nil
	}

	return cacheKeyFromConfig(p.config).String(), true, nil
}

func (p *puller) Snapshot(ctx context.Context) (cache.ImmutableRef, error) {
	p.resolveLocal()
	if err := p.resolve(ctx); err != nil {
		return nil, err
	}

	if p.config != nil {
		img, err := p.is.ImageStore.Get(image.ID(digest.FromBytes(p.config)))
		if err == nil {
			if len(img.RootFS.DiffIDs) == 0 {
				return nil, nil
			}
			ref, err := p.is.CacheAccessor.GetFromSnapshotter(ctx, string(img.RootFS.ChainID()), cache.WithDescription(fmt.Sprintf("from local %s", p.ref)))
			if err != nil {
				return nil, err
			}
			return ref, nil
		}
	}

	ongoing := newJobs(p.ref)

	pctx, stopProgress := context.WithCancel(ctx)

	pw, _, ctx := progress.FromContext(ctx)
	defer pw.Close()

	progressDone := make(chan struct{})
	go func() {
		showProgress(pctx, ongoing, p.is.ContentStore, pw)
		close(progressDone)
	}()
	defer func() {
		<-progressDone
	}()

	fetcher, err := p.resolver.Fetcher(ctx, p.ref)
	if err != nil {
		stopProgress()
		return nil, err
	}

	var (
		schema1Converter *schema1.Converter
		handlers         []images.Handler
	)
	if p.desc.MediaType == images.MediaTypeDockerSchema1Manifest {
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
			ongoing.add(desc)
			return nil, nil
		}))

		// Get all the children for a descriptor
		childrenHandler := images.ChildrenHandler(p.is.ContentStore)
		// Set any children labels for that content
		childrenHandler = images.SetChildrenLabels(p.is.ContentStore, childrenHandler)
		// Filter the children by the platform
		childrenHandler = images.FilterPlatforms(childrenHandler, platforms.Default())

		handlers = append(handlers,
			remotes.FetchHandler(p.is.ContentStore, fetcher),
			childrenHandler,
		)
	}

	if err := images.Dispatch(ctx, images.Handlers(handlers...), p.desc); err != nil {
		stopProgress()
		return nil, err
	}
	defer stopProgress()

	if schema1Converter != nil {
		p.desc, err = schema1Converter.Convert(ctx)
		if err != nil {
			return nil, err
		}
	}

	mfst, err := images.Manifest(ctx, p.is.ContentStore, p.desc, platforms.Default())
	if err != nil {
		return nil, err
	}

	config, err := images.Config(ctx, p.is.ContentStore, p.desc, platforms.Default())
	if err != nil {
		return nil, err
	}

	dt, err := content.ReadBlob(ctx, p.is.ContentStore, config)
	if err != nil {
		return nil, err
	}

	var img ocispec.Image
	if err := json.Unmarshal(dt, &img); err != nil {
		return nil, err
	}

	if len(mfst.Layers) != len(img.RootFS.DiffIDs) {
		return nil, errors.Errorf("invalid config for manifest")
	}

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
					pw.Write("extracting "+p.ID, progress.Status{
						Action:    "extract",
						Started:   &st.st,
						Completed: end,
					})
				}
			}
		}
	}()

	if len(mfst.Layers) == 0 {
		return nil, nil
	}

	layers := make([]xfer.DownloadDescriptor, 0, len(mfst.Layers))

	for i, desc := range mfst.Layers {
		ongoing.add(desc)
		layers = append(layers, &layerDescriptor{
			desc:    desc,
			diffID:  layer.DiffID(img.RootFS.DiffIDs[i]),
			fetcher: fetcher,
			ref:     p.src.Reference,
			is:      p.is,
		})
	}

	defer func() {
		<-progressDone
		for _, desc := range mfst.Layers {
			p.is.ContentStore.Delete(context.TODO(), desc.Digest)
		}
	}()

	r := image.NewRootFS()
	rootFS, release, err := p.is.DownloadManager.Download(ctx, *r, runtime.GOOS, layers, pkgprogress.ChanOutput(pchan))
	if err != nil {
		return nil, err
	}
	stopProgress()

	ref, err := p.is.CacheAccessor.GetFromSnapshotter(ctx, string(rootFS.ChainID()), cache.WithDescription(fmt.Sprintf("pulled from %s", p.ref)))
	release()
	if err != nil {
		return nil, err
	}

	// TODO: handle windows layers for cross platform builds

	if p.src.RecordType != "" && cache.GetRecordType(ref) == "" {
		if err := cache.SetRecordType(ref, p.src.RecordType); err != nil {
			ref.Release(context.TODO())
			return nil, err
		}
	}

	return ref, nil
}

// Fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error)
type layerDescriptor struct {
	is      *imageSource
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

func showProgress(ctx context.Context, ongoing *jobs, cs content.Store, pw progress.Writer) {
	var (
		ticker   = time.NewTicker(100 * time.Millisecond)
		statuses = map[string]statusInfo{}
		done     bool
	)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			done = true
		}

		resolved := "resolved"
		if !ongoing.isResolved() {
			resolved = "resolving"
		}
		statuses[ongoing.name] = statusInfo{
			Ref:    ongoing.name,
			Status: resolved,
		}

		actives := make(map[string]statusInfo)

		if !done {
			active, err := cs.ListStatuses(ctx)
			if err != nil {
				// log.G(ctx).WithError(err).Error("active check failed")
				continue
			}
			// update status of active entries!
			for _, active := range active {
				actives[active.Ref] = statusInfo{
					Ref:       active.Ref,
					Status:    "downloading",
					Offset:    active.Offset,
					Total:     active.Total,
					StartedAt: active.StartedAt,
					UpdatedAt: active.UpdatedAt,
				}
			}
		}

		// now, update the items in jobs that are not in active
		for _, j := range ongoing.jobs() {
			refKey := remotes.MakeRefKey(ctx, j.Descriptor)
			if a, ok := actives[refKey]; ok {
				started := j.started
				pw.Write(j.Digest.String(), progress.Status{
					Action:  a.Status,
					Total:   int(a.Total),
					Current: int(a.Offset),
					Started: &started,
				})
				continue
			}

			if !j.done {
				info, err := cs.Info(context.TODO(), j.Digest)
				if err != nil {
					if errdefs.IsNotFound(err) {
						// pw.Write(j.Digest.String(), progress.Status{
						// 	Action: "waiting",
						// })
						continue
					}
				} else {
					j.done = true
				}

				if done || j.done {
					started := j.started
					createdAt := info.CreatedAt
					pw.Write(j.Digest.String(), progress.Status{
						Action:    "done",
						Current:   int(info.Size),
						Total:     int(info.Size),
						Completed: &createdAt,
						Started:   &started,
					})
				}
			}
		}
		if done {
			return
		}
	}
}

// jobs provides a way of identifying the download keys for a particular task
// encountering during the pull walk.
//
// This is very minimal and will probably be replaced with something more
// featured.
type jobs struct {
	name     string
	added    map[digest.Digest]*job
	mu       sync.Mutex
	resolved bool
}

type job struct {
	ocispec.Descriptor
	done    bool
	started time.Time
}

func newJobs(name string) *jobs {
	return &jobs{
		name:  name,
		added: make(map[digest.Digest]*job),
	}
}

func (j *jobs) add(desc ocispec.Descriptor) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if _, ok := j.added[desc.Digest]; ok {
		return
	}
	j.added[desc.Digest] = &job{
		Descriptor: desc,
		started:    time.Now(),
	}
}

func (j *jobs) jobs() []*job {
	j.mu.Lock()
	defer j.mu.Unlock()

	descs := make([]*job, 0, len(j.added))
	for _, j := range j.added {
		descs = append(descs, j)
	}
	return descs
}

func (j *jobs) isResolved() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.resolved
}

type statusInfo struct {
	Ref       string
	Status    string
	Offset    int64
	Total     int64
	StartedAt time.Time
	UpdatedAt time.Time
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

// cacheKeyFromConfig returns a stable digest from image config. If image config
// is a known oci image we will use chainID of layers.
func cacheKeyFromConfig(dt []byte) digest.Digest {
	var img ocispec.Image
	err := json.Unmarshal(dt, &img)
	if err != nil {
		return digest.FromBytes(dt)
	}
	if img.RootFS.Type != "layers" {
		return digest.FromBytes(dt)
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
