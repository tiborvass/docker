package images // import "github.com/tiborvass/docker/daemon/images"

import (
	"github.com/tiborvass/docker/builder"
	"github.com/tiborvass/docker/image/cache"
	"github.com/sirupsen/logrus"
)

// MakeImageCache creates a stateful image cache.
func (i *ImageService) MakeImageCache(sourceRefs []string) builder.ImageCache {
	if len(sourceRefs) == 0 {
		return cache.NewLocal(i.imageStore)
	}

	cache := cache.New(i.imageStore)

	for _, ref := range sourceRefs {
		img, err := i.GetImage(ref, nil)
		if err != nil {
			logrus.Warnf("Could not look up %s for cache resolution, skipping: %+v", ref, err)
			continue
		}
		cache.Populate(img)
	}

	return cache
}
