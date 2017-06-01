package remotecontext

import (
	"os"

	"github.com/tiborvass/docker/builder"
	"github.com/tiborvass/docker/builder/remotecontext/git"
	"github.com/tiborvass/docker/pkg/archive"
)

// MakeGitContext returns a Context from gitURL that is cloned in a temporary directory.
func MakeGitContext(gitURL string) (builder.Source, error) {
	root, err := git.Clone(gitURL)
	if err != nil {
		return nil, err
	}

	c, err := archive.Tar(root, archive.Uncompressed)
	if err != nil {
		return nil, err
	}

	defer func() {
		// TODO: print errors?
		c.Close()
		os.RemoveAll(root)
	}()
	return MakeTarSumContext(c)
}
