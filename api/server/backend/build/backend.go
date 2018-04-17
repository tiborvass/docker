package build // import "github.com/tiborvass/docker/api/server/backend/build"

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/backend"
	"github.com/tiborvass/docker/builder"
	buildkit "github.com/tiborvass/docker/builder/builder-next"
	"github.com/tiborvass/docker/builder/fscache"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/pkg/errors"
)

// ImageComponent provides an interface for working with images
type ImageComponent interface {
	SquashImage(from string, to string) (string, error)
	TagImageWithReference(image.ID, reference.Named) error
}

// Builder defines interface for running a build
type Builder interface {
	Build(context.Context, backend.BuildConfig) (*builder.Result, error)
}

// Backend provides build functionality to the API router
type Backend struct {
	builder        Builder
	fsCache        *fscache.FSCache
	imageComponent ImageComponent
	buildkit       *buildkit.Builder
}

// NewBackend creates a new build backend from components
func NewBackend(components ImageComponent, builder Builder, fsCache *fscache.FSCache, buildkit *buildkit.Builder) (*Backend, error) {
	return &Backend{imageComponent: components, builder: builder, fsCache: fsCache, buildkit: buildkit}, nil
}

// Build builds an image from a Source
func (b *Backend) Build(ctx context.Context, config backend.BuildConfig) (string, error) {
	options := config.Options
	useBuildKit := false
	if strings.HasPrefix(options.SessionID, "buildkit:") {
		useBuildKit = true
		options.SessionID = strings.TrimPrefix(options.SessionID, "buildkit:")
	}

	tagger, err := NewTagger(b.imageComponent, config.ProgressWriter.StdoutFormatter, options.Tags)
	if err != nil {
		return "", err
	}

	var build *builder.Result
	if useBuildKit {
		build, err = b.buildkit.Build(ctx, config)
		if err != nil {
			return "", err
		}
	} else {
		build, err = b.builder.Build(ctx, config)
		if err != nil {
			return "", err
		}
	}

	var imageID = build.ImageID
	if options.Squash {
		if imageID, err = squashBuild(build, b.imageComponent); err != nil {
			return "", err
		}
		if config.ProgressWriter.AuxFormatter != nil {
			if err = config.ProgressWriter.AuxFormatter.Emit(types.BuildResult{ID: imageID}); err != nil {
				return "", err
			}
		}
	}

	stdout := config.ProgressWriter.StdoutFormatter
	fmt.Fprintf(stdout, "Successfully built %s\n", stringid.TruncateID(imageID))
	err = tagger.TagImages(image.ID(imageID))
	return imageID, err
}

// PruneCache removes all cached build sources
func (b *Backend) PruneCache(ctx context.Context) (*types.BuildCachePruneReport, error) {
	size, err := b.fsCache.Prune(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prune build cache")
	}
	return &types.BuildCachePruneReport{SpaceReclaimed: size}, nil
}

func squashBuild(build *builder.Result, imageComponent ImageComponent) (string, error) {
	var fromID string
	if build.FromImage != nil {
		fromID = build.FromImage.ImageID()
	}
	imageID, err := imageComponent.SquashImage(build.ImageID, fromID)
	if err != nil {
		return "", errors.Wrap(err, "error squashing image")
	}
	return imageID, nil
}
