package daemon

import (
	"fmt"
	"time"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/layer"
	"github.com/tiborvass/docker/reference"
)

// LookupImage looks up an image by name and returns it as an ImageInspect
// structure.
func (daemon *Daemon) LookupImage(name string) (*types.ImageInspect, error) {
	img, err := daemon.GetImage(name)
	if err != nil {
		return nil, fmt.Errorf("No such image: %s", name)
	}

	refs := daemon.referenceStore.References(img.ID())
	repoTags := []string{}
	repoDigests := []string{}
	for _, ref := range refs {
		switch ref.(type) {
		case reference.NamedTagged:
			repoTags = append(repoTags, ref.String())
		case reference.Canonical:
			repoDigests = append(repoDigests, ref.String())
		}
	}

	var size int64
	var layerMetadata map[string]string
	layerID := img.RootFS.ChainID()
	if layerID != "" {
		l, err := daemon.layerStore.Get(layerID)
		if err != nil {
			return nil, err
		}
		defer layer.ReleaseAndLog(daemon.layerStore, l)
		size, err = l.Size()
		if err != nil {
			return nil, err
		}

		layerMetadata, err = l.Metadata()
		if err != nil {
			return nil, err
		}
	}

	comment := img.Comment
	if len(comment) == 0 && len(img.History) > 0 {
		comment = img.History[len(img.History)-1].Comment
	}

	imageInspect := &types.ImageInspect{
		ID:              img.ID().String(),
		RepoTags:        repoTags,
		RepoDigests:     repoDigests,
		Parent:          img.Parent.String(),
		Comment:         comment,
		Created:         img.Created.Format(time.RFC3339Nano),
		Container:       img.Container,
		ContainerConfig: &img.ContainerConfig,
		DockerVersion:   img.DockerVersion,
		Author:          img.Author,
		Config:          img.Config,
		Architecture:    img.Architecture,
		Os:              img.OS,
		Size:            size,
		VirtualSize:     size, // TODO: field unused, deprecate
		RootFS:          rootFSToAPIType(img.RootFS),
	}

	imageInspect.GraphDriver.Name = daemon.GraphDriverName()

	imageInspect.GraphDriver.Data = layerMetadata

	return imageInspect, nil
}
