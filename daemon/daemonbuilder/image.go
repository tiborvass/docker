package daemonbuilder

import (
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/runconfig"
)

type imgWrap struct {
	inner *image.Image
}

func (img imgWrap) ID() string {
	return string(img.inner.ID())
}

func (img imgWrap) Config() *runconfig.Config {
	return img.inner.Config
}
