package images // import "github.com/docker/docker/daemon/images"

import (
	"context"
	"fmt"
	"sync"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/docker/distribution/digestset"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/builder"
	buildcache "github.com/docker/docker/image/cache"
	"github.com/docker/docker/layer"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

type cachedImage struct {
	config ocispec.Descriptor
	parent digest.Digest

	// Mutable values
	m          sync.Mutex
	references []reference.Named
	children   []digest.Digest

	// Layer held by Docker, this should get removed when
	// moved to containerd snapshotters. The garbage
	// collection in containerd is reasonable for cleanup.
	layer layer.Layer
}

type cache struct {
	m           sync.RWMutex
	ids         *digestset.Set
	targets     *digestset.Set
	descriptors map[digest.Digest]ocispec.Descriptor
	layers      map[string]map[digest.Digest]layer.Layer

	// idCache maps Docker identifiers
	// deprecated
	idCache map[digest.Digest]*cachedImage
	// tCache maps target digests to images
	// deprecated
	tCache map[digest.Digest]*cachedImage
}

func (c *cache) byID(id digest.Digest) *cachedImage {
	c.m.RLock()
	img, ok := c.idCache[id]
	c.m.RUnlock()
	if !ok {
		return nil
	}
	return img
}

func (c *cache) byTarget(target digest.Digest) *cachedImage {
	c.m.RLock()
	img, ok := c.tCache[target]
	c.m.RUnlock()
	if !ok {
		return nil
	}
	return img
}

// LoadCache loads the image cache by scanning containerd images
// and listening to containerd events
// This process can be backgrounded and will hold hold the cache
// lock until fully populated.
func (i *ImageService) LoadCache(ctx context.Context) error {
	namespace, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return err
	}
	log.G(ctx).WithField("namespace", namespace).Debugf("loading cache")

	_, err = i.loadNSCache(ctx, namespace)
	return err
}

func (i *ImageService) loadNSCache(ctx context.Context, namespace string) (*cache, error) {
	i.cacheL.Lock()
	defer i.cacheL.Unlock()

	var (
		cs = i.client.ContentStore()
		is = i.client.ImageService()
		c  = &cache{
			targets:     digestset.NewSet(),
			descriptors: map[digest.Digest]ocispec.Descriptor{},
			layers:      map[string]map[digest.Digest]layer.Layer{},

			// Deprecated
			ids:     digestset.NewSet(),
			idCache: map[digest.Digest]*cachedImage{},
			tCache:  map[digest.Digest]*cachedImage{},
		}
	)

	// Load layers
	for _, backend := range i.layerBackends {
		backendCache := map[digest.Digest]layer.Layer{}
		name := backend.DriverName()
		label := fmt.Sprintf("%s%s", LabelLayerPrefix, name)
		err := cs.Walk(ctx, func(info content.Info) error {
			value := digest.Digest(info.Labels[label])
			if _, ok := backendCache[value]; ok {
				return nil
			}
			l, err := backend.Get(layer.ChainID(value))
			if err != nil {
				log.G(ctx).WithError(err).WithField("digest", info.Digest).WithField("driver", name).Warnf("unable to get layer")
			} else {
				log.G(ctx).WithField("digest", info.Digest).WithField("driver", name).Debugf("retaining layer %s", value)
				backendCache[value] = l
			}
			return nil
		}, fmt.Sprintf("labels.%q", label))
		if err != nil {
			return nil, err
		}

		c.layers[name] = backendCache
	}

	// TODO(containerd): This must use some streaming approach
	imgs, err := is.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, img := range imgs {
		var (
			named reference.Named
			id    ocispec.Descriptor
		)

		if danglingID, ok := img.Labels[LabelImageDangling]; !ok {
			ref, err := reference.Parse(img.Name)
			if err != nil {
				log.G(ctx).WithError(err).WithField("name", img.Name).Debug("skipping invalid image name")
				continue
			}
			var ok bool
			named, ok = ref.(reference.Named)
			if !ok {
				log.G(ctx).WithField("name", img.Name).Debug("skipping invalid image name with no name component")
				continue
			}
		} else {
			dgst, err := digest.Parse(danglingID)
			if err != nil {
				log.G(ctx).WithError(err).WithField("id", danglingID).Debug("skipping invalid image id label (dangling)")
				continue
			}
			id = ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageConfig,
				Digest:    dgst,
			}
		}

		ci := c.tCache[img.Target.Digest]
		if ci == nil {
			if img.Target.MediaType == images.MediaTypeDockerSchema2Config || img.Target.MediaType == ocispec.MediaTypeImageConfig {
				id = img.Target
			}
			if id.Digest == "" {
				idstr, ok := img.Labels[LabelImageID]
				if !ok {
					cs := i.client.ContentStore()
					// TODO(containerd): resolve architecture from context
					// TODO(containerd): support multi-platform images
					platform := platforms.Default()
					desc, err := images.Config(ctx, cs, img.Target, platform)
					if err != nil {
						log.G(ctx).WithError(err).WithField("name", img.Name).Debug("unable to resolve image config for platform")
						continue
					}
					id = desc
				} else {
					dgst, err := digest.Parse(idstr)
					if err != nil {
						log.G(ctx).WithError(err).WithField("name", img.Name).Debug("skipping invalid image id label")
						continue
					}
					id = ocispec.Descriptor{
						MediaType: ocispec.MediaTypeImageConfig,
						Digest:    dgst,
					}
				}
			}

			ci = c.idCache[id.Digest]
			if ci == nil {
				ci = &cachedImage{
					config: id,
				}
				if s := img.Labels[LabelImageParent]; s != "" {
					pid, err := digest.Parse(s)
					if err != nil {
						log.G(ctx).WithError(err).WithField("name", img.Name).Debug("skipping invalid parent label")
					} else {
						ci.parent = pid
					}
				}
				//diffIDs, err := images.RootFS(ctx, i.client.ContentStore(), ci.config)
				//if err != nil {
				//	log.G(ctx).WithError(err).WithField("name", img.Name).Debug("unable to load image rootfs")
				//	continue
				//}

				//// TODO(containerd): choose correct platform
				//ci.layer, err = i.backends[0].Get(layer.ChainID(identity.ChainID(diffIDs)))
				//if err != nil {
				//	log.G(ctx).WithError(err).WithField("name", img.Name).Debug("no layer for image")
				//	continue
				//}

				c.idCache[id.Digest] = ci
				c.ids.Add(id.Digest)
			}
			c.tCache[img.Target.Digest] = ci
			c.targets.Add(img.Target.Digest)
			c.descriptors[img.Target.Digest] = img.Target

			// Load image layer to prevent removal
		}
		if named != nil {
			ci.addReference(named)
		}
	}
	i.cache[namespace] = c

	return c, nil
}

func (ci *cachedImage) addReference(named reference.Named) {
	var (
		i int
		s = named.String()
	)

	// Add references, add in sorted place
	for ; i < len(ci.references); i++ {
		if rs := ci.references[i].String(); s < rs {
			ci.references = append(ci.references, nil)
			copy(ci.references[i+1:], ci.references[i:])
			ci.references[i] = named
			break
		} else if rs == s {
			break
		}
	}
	if i == len(ci.references) {
		ci.references = append(ci.references, named)
	}
}

func (ci *cachedImage) addChild(d digest.Digest) {
	var i int

	// Add references, add in sorted place
	for ; i < len(ci.children); i++ {
		if d < ci.children[i] {
			ci.children = append(ci.children, "")
			copy(ci.children[i+1:], ci.children[i:])
			ci.children[i] = d
			break
		} else if ci.children[i] == d {
			break
		}
	}
	if i == len(ci.children) {
		ci.children = append(ci.children, d)
	}
}

func (i *ImageService) getCache(ctx context.Context) (c *cache, err error) {
	namespace, ok := namespaces.Namespace(ctx)
	if !ok {
		namespace = i.namespace
	}
	i.cacheL.RLock()
	c, ok = i.cache[namespace]
	i.cacheL.RUnlock()
	if !ok {
		c, err = i.loadNSCache(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// MakeImageCache creates a stateful image cache for build.
func (i *ImageService) MakeImageCache(sourceRefs []string) builder.ImageCache {
	if len(sourceRefs) == 0 {
		return buildcache.NewLocal(i.imageStore)
	}

	cache := buildcache.New(i.imageStore)

	for _, ref := range sourceRefs {
		img, err := i.getDockerImage(ref)
		if err != nil {
			logrus.Warnf("Could not look up %s for cache resolution, skipping: %+v", ref, err)
			continue
		}
		cache.Populate(img)
	}

	return cache
}
