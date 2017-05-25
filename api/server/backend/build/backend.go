package build

import (
	"fmt"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/types/backend"
	"github.com/tiborvass/docker/builder"
	"github.com/tiborvass/docker/builder/dockerfile"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/pkg/idtools"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// ImageComponent provides an interface for working with images
type ImageComponent interface {
	SquashImage(from string, to string) (string, error)
	TagImageWithReference(image.ID, reference.Named) error
}

// Backend provides build functionality to the API router
type Backend struct {
	manager        *dockerfile.BuildManager
	imageComponent ImageComponent
}

// NewBackend creates a new build backend from components
func NewBackend(components ImageComponent, builderBackend builder.Backend, idMappings *idtools.IDMappings) *Backend {
	manager := dockerfile.NewBuildManager(builderBackend, idMappings)
	return &Backend{imageComponent: components, manager: manager}
}

// Build builds an image from a Source
func (b *Backend) Build(ctx context.Context, config backend.BuildConfig) (string, error) {
	options := config.Options
	tagger, err := NewTagger(b.imageComponent, config.ProgressWriter.StdoutFormatter, options.Tags)
	if err != nil {
		return "", err
	}

	build, err := b.manager.Build(ctx, config)
	if err != nil {
		return "", err
	}

	var imageID = build.ImageID
	if options.Squash {
		if imageID, err = squashBuild(build, b.imageComponent); err != nil {
			return "", err
		}
	}

	stdout := config.ProgressWriter.StdoutFormatter
	fmt.Fprintf(stdout, "Successfully built %s\n", stringid.TruncateID(imageID))
	err = tagger.TagImages(image.ID(imageID))
	return imageID, err
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
