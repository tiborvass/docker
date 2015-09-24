package daemon

import (
	"github.com/tiborvass/docker/context"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/runconfig"
)

// ContainerCommitConfig contains build configs for commit operation,
// and is used when making a commit with the current state of the container.
type ContainerCommitConfig struct {
	Pause   bool
	Repo    string
	Tag     string
	Author  string
	Comment string
	Config  *runconfig.Config
}

// Commit creates a new filesystem image from the current state of a container.
// The image can optionally be tagged into a repository.
func (daemon *Daemon) Commit(ctx context.Context, container *Container, c *ContainerCommitConfig) (*image.Image, error) {
	if c.Pause && !container.isPaused() {
		container.pause(ctx)
		defer container.unpause(ctx)
	}

	rwTar, err := container.exportContainerRw()
	if err != nil {
		return nil, err
	}
	defer func() {
		if rwTar != nil {
			rwTar.Close()
		}
	}()

	// Create a new image from the container's base layers + a new layer from container changes
	img, err := daemon.graph.Create(rwTar, container.ID, container.ImageID, c.Comment, c.Author, container.Config, c.Config)
	if err != nil {
		return nil, err
	}

	// Register the image if needed
	if c.Repo != "" {
		if err := daemon.repositories.Tag(c.Repo, c.Tag, img.ID, true); err != nil {
			return img, err
		}
	}
	container.logEvent(ctx, "commit")
	return img, nil
}
