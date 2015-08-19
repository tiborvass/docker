package server

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/builder"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/daemon"
	"github.com/docker/docker/graph"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/chrootarchive"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/parsers"
	"github.com/docker/docker/pkg/system"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/runconfig"
)

type builderDocker struct {
	*daemon.Daemon
	OutOld      io.Writer
	AuthConfigs map[string]cliconfig.AuthConfig
}

// ensure builderDocker implements builder.Docker
var _ builder.Docker = builderDocker{}

func (d builderDocker) LookupImage(name string) (*image.Image, error) {
	img, err := d.Daemon.Repositories().LookupImage(name)
	logrus.Debugf("lookupimage: %s: %v", name, err)
	return img, err
}

func (d builderDocker) Pull(name string) (*image.Image, error) {
	remote, tag := parsers.ParseRepositoryTag(name)
	if tag == "" {
		tag = "latest"
	}

	pullRegistryAuth := &cliconfig.AuthConfig{}
	if len(d.AuthConfigs) > 0 {
		// The request came with a full auth config file, we prefer to use that
		repoInfo, err := d.Daemon.RegistryService.ResolveRepository(remote)
		if err != nil {
			return nil, err
		}

		resolvedConfig := registry.ResolveAuthConfig(
			&cliconfig.ConfigFile{AuthConfigs: d.AuthConfigs},
			repoInfo.Index,
		)
		pullRegistryAuth = &resolvedConfig
	}

	imagePullConfig := &graph.ImagePullConfig{
		AuthConfig: pullRegistryAuth,
		OutStream:  ioutils.NopWriteCloser(d.OutOld),
	}

	if err := d.Daemon.Repositories().Pull(remote, tag, imagePullConfig); err != nil {
		return nil, err
	}

	return d.Daemon.Repositories().LookupImage(name)
}

func (d builderDocker) Container(id string) (*daemon.Container, error) {
	return d.Daemon.Get(id)
}

func (d builderDocker) Create(cfg *runconfig.Config, hostCfg *runconfig.HostConfig) (*daemon.Container, error) {
	// TODO: what to do with warning (second output) ?
	c, _, err := d.Daemon.Create(cfg, hostCfg, "")
	if err != nil {
		return nil, err
	}
	return c, c.Mount()
}

func (d builderDocker) Remove(id string, cfg *daemon.ContainerRmConfig) error {
	return d.Daemon.ContainerRm(id, cfg)
}

func (d builderDocker) Commit(c *daemon.Container, cfg *daemon.ContainerCommitConfig) (*image.Image, error) {
	return d.Daemon.Commit(c, cfg)
}

func (d builderDocker) Copy(c *daemon.Container, destPath, srcPath string) (err error) {
	var destExists = true

	// Work in daemon-local OS specific file paths
	destPath = filepath.FromSlash(destPath)

	dest, err := c.GetResourcePath(destPath)
	if err != nil {
		return err
	}

	// Preserve the trailing slash
	if strings.HasSuffix(destPath, string(os.PathSeparator)) || destPath == "." {
		dest += string(os.PathSeparator)
	}

	destPath = dest

	destStat, err := os.Stat(destPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logrus.Errorf("Error performing os.Stat on %s. %s", destPath, err)
			return err
		}
		destExists = false
	}

	fi, err := os.Stat(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: no such file or directory", srcPath)
		}
		return err
	}

	if fi.IsDir() {
		// copy as directory
		if err := chrootarchive.CopyWithTar(srcPath, destPath); err != nil {
			return err
		}
		return fixPermissions(srcPath, destPath, 0, 0, destExists)
	}

	if err := system.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	if err := chrootarchive.CopyFileWithTar(srcPath, destPath); err != nil {
		return err
	}

	if destExists && destStat.IsDir() {
		destPath = filepath.Join(destPath, filepath.Base(srcPath))
	}

	return fixPermissions(srcPath, destPath, 0, 0, destExists)
}

func (d builderDocker) GetCachedImage(imgID builder.ImageID, cfg *runconfig.Config) (builder.ImageID, error) {
	cache, err := d.Daemon.ImageGetCached(string(imgID), cfg)
	if cache == nil || err != nil {
		return "", err
	}
	return builder.ImageID(cache.ID), nil
}

func (d builderDocker) Cache(imgID builder.ImageID, cfg *runconfig.Config) error {
	return nil
}
