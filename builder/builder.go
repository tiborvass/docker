package builder

import (
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/daemon"
	"github.com/docker/docker/image"
	"github.com/docker/docker/runconfig"
)

type ImageID string

type Builder interface {
	// TODO: make this return a reference and remove ImageID
	Build(Docker, Context) ImageID
}

type ImageCache interface {
	GetCachedImage(ImageID, *runconfig.Config) (ImageID, error)
	Cache(ImageID, *runconfig.Config) error
}

type Docker interface {
	// use digest reference instead of name
	LookupImage(name string) (*image.Image, error)
	Pull(name string) (*image.Image, error)

	// move daemon.Container to its own package
	Container(id string) (*daemon.Container, error)
	Create(*runconfig.Config, *runconfig.HostConfig) (*daemon.Container, error)
	Remove(id string, cfg *daemon.ContainerRmConfig) error
	Commit(*daemon.Container, *daemon.ContainerCommitConfig) (*image.Image, error)
	Copy(c *daemon.Container, destPath, srcPath string) error
}

type ProtoContext interface {
	Get(path string) (ContextEntry, error)
	Walk(filepath.WalkFunc) error
}

type Context interface {
	ProtoContext
	Open(path string) (io.ReadCloser, error)
}

type HelperContext struct {
	ProtoContext
}

func (c HelperContext) Open(path string) (io.ReadCloser, error) {
	e, err := c.ProtoContext.Get(path)
	if err != nil {
		return nil, err
	}
	return e.Open()
}

type ContextEntry interface {
	os.FileInfo
	Path() string
	Open() (io.ReadCloser, error)
}

type FileInfo struct {
	os.FileInfo
	FilePath string
}

func (fi FileInfo) Path() string {
	return fi.FilePath
}

func (fi FileInfo) Open() (io.ReadCloser, error) {
	return os.Open(fi.FilePath)
}

func (fi FileInfo) Stat() os.FileInfo {
	return fi.FileInfo
}

type HashFileInfo struct {
	FileInfo
	Hash string
}
