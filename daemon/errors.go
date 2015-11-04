package daemon

import (
	"strings"

	derr "github.com/tiborvass/docker/errors"
	"github.com/tiborvass/docker/graph/tags"
	"github.com/tiborvass/docker/pkg/parsers"
)

func (d *Daemon) graphNotExistToErrcode(imageName string, err error) error {
	if d.Graph().IsNotExist(err, imageName) {
		if strings.Contains(imageName, "@") {
			return derr.ErrorCodeNoSuchImageHash.WithArgs(imageName)
		}
		img, tag := parsers.ParseRepositoryTag(imageName)
		if tag == "" {
			tag = tags.DefaultTag
		}
		return derr.ErrorCodeNoSuchImageTag.WithArgs(img, tag)
	}
	return err
}
