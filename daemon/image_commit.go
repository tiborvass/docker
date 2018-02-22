package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"encoding/json"
	"io"

	"github.com/tiborvass/docker/api/types/backend"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/layer"
	"github.com/tiborvass/docker/pkg/ioutils"
	"github.com/tiborvass/docker/pkg/system"
)

func (daemon *Daemon) commitImage(c backend.CommitConfig) (image.ID, error) {
	layerStore, ok := daemon.layerStores[c.ContainerOS]
	if !ok {
		return "", system.ErrNotSupportedOperatingSystem
	}
	rwTar, err := exportContainerRw(layerStore, c.ContainerID, c.ContainerMountLabel)
	if err != nil {
		return "", err
	}
	defer func() {
		if rwTar != nil {
			rwTar.Close()
		}
	}()

	var parent *image.Image
	if c.ParentImageID == "" {
		parent = new(image.Image)
		parent.RootFS = image.NewRootFS()
	} else {
		parent, err = daemon.imageStore.Get(image.ID(c.ParentImageID))
		if err != nil {
			return "", err
		}
	}

	l, err := layerStore.Register(rwTar, parent.RootFS.ChainID())
	if err != nil {
		return "", err
	}
	defer layer.ReleaseAndLog(layerStore, l)

	cc := image.ChildConfig{
		ContainerID:     c.ContainerID,
		Author:          c.Author,
		Comment:         c.Comment,
		ContainerConfig: c.ContainerConfig,
		Config:          c.Config,
		DiffID:          l.DiffID(),
	}
	config, err := json.Marshal(image.NewChildImage(parent, cc, c.ContainerOS))
	if err != nil {
		return "", err
	}

	id, err := daemon.imageStore.Create(config)
	if err != nil {
		return "", err
	}

	if c.ParentImageID != "" {
		if err := daemon.imageStore.SetParent(id, image.ID(c.ParentImageID)); err != nil {
			return "", err
		}
	}
	return id, nil
}

func exportContainerRw(layerStore layer.Store, id, mountLabel string) (arch io.ReadCloser, err error) {
	rwlayer, err := layerStore.GetRWLayer(id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			layerStore.ReleaseRWLayer(rwlayer)
		}
	}()

	// TODO: this mount call is not necessary as we assume that TarStream() should
	// mount the layer if needed. But the Diff() function for windows requests that
	// the layer should be mounted when calling it. So we reserve this mount call
	// until windows driver can implement Diff() interface correctly.
	_, err = rwlayer.Mount(mountLabel)
	if err != nil {
		return nil, err
	}

	archive, err := rwlayer.TarStream()
	if err != nil {
		rwlayer.Unmount()
		return nil, err
	}
	return ioutils.NewReadCloserWrapper(archive, func() error {
			archive.Close()
			err = rwlayer.Unmount()
			layerStore.ReleaseRWLayer(rwlayer)
			return err
		}),
		nil
}

// CommitBuildStep is used by the builder to create an image for each step in
// the build.
//
// This method is different from CreateImageFromContainer:
//   * it doesn't attempt to validate container state
//   * it doesn't send a commit action to metrics
//   * it doesn't log a container commit event
//
// This is a temporary shim. Should be removed when builder stops using commit.
func (daemon *Daemon) CommitBuildStep(c backend.CommitConfig) (image.ID, error) {
	container, err := daemon.GetContainer(c.ContainerID)
	if err != nil {
		return "", err
	}
	c.ContainerMountLabel = container.MountLabel
	c.ContainerOS = container.OS
	c.ParentImageID = string(container.ImageID)
	return daemon.commitImage(c)
}
