package builder

// internals for handling commands. Covers many areas and a lot of
// non-contiguous functionality. Please read the comments.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/tiborvass/docker/archive"
	"github.com/tiborvass/docker/daemon"
	imagepkg "github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/pkg/log"
	"github.com/tiborvass/docker/pkg/parsers"
	"github.com/tiborvass/docker/pkg/symlink"
	"github.com/tiborvass/docker/pkg/system"
	"github.com/tiborvass/docker/pkg/tarsum"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/utils"
)

func (b *BuildFile) readContext(context io.Reader) error {
	tmpdirPath, err := ioutil.TempDir("", "docker-build")
	if err != nil {
		return err
	}

	decompressedStream, err := archive.DecompressStream(context)
	if err != nil {
		return err
	}

	b.context = &tarsum.TarSum{Reader: decompressedStream, DisableCompression: true}
	if err := archive.Untar(b.context, tmpdirPath, nil); err != nil {
		return err
	}

	b.contextPath = tmpdirPath
	return nil
}

func (b *BuildFile) commit(id string, autoCmd []string, comment string) error {
	if b.image == "" {
		return fmt.Errorf("Please provide a source image with `from` prior to commit")
	}
	b.Config.Image = b.image
	if id == "" {
		cmd := b.Config.Cmd
		b.Config.Cmd = []string{"/bin/sh", "-c", "#(nop) " + comment}
		defer func(cmd []string) { b.Config.Cmd = cmd }(cmd)

		hit, err := b.probeCache()
		if err != nil {
			return err
		}
		if hit {
			return nil
		}

		container, warnings, err := b.Options.Daemon.Create(b.Config, "")
		if err != nil {
			return err
		}
		for _, warning := range warnings {
			fmt.Fprintf(b.Options.OutStream, " ---> [Warning] %s\n", warning)
		}
		b.TmpContainers[container.ID] = struct{}{}
		fmt.Fprintf(b.Options.OutStream, " ---> Running in %s\n", utils.TruncateID(container.ID))
		id = container.ID

		if err := container.Mount(); err != nil {
			return err
		}
		defer container.Unmount()
	}
	container := b.Options.Daemon.Get(id)
	if container == nil {
		return fmt.Errorf("An error occured while creating the container")
	}

	// Note: Actually copy the struct
	autoConfig := *b.Config
	autoConfig.Cmd = autoCmd
	// Commit the container
	image, err := b.Options.Daemon.Commit(container, "", "", "", b.maintainer, true, &autoConfig)
	if err != nil {
		return err
	}
	b.TmpImages[image.ID] = struct{}{}
	b.image = image.ID
	return nil
}

func (b *BuildFile) runContextCommand(args []string, allowRemote bool, allowDecompression bool, cmdName string) error {
	if b.context == nil {
		return fmt.Errorf("No context given. Impossible to use %s", cmdName)
	}

	if len(args) != 2 {
		return fmt.Errorf("Invalid %s format", cmdName)
	}

	orig := args[0]
	dest := args[1]

	cmd := b.Config.Cmd
	b.Config.Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("#(nop) %s %s in %s", cmdName, orig, dest)}
	defer func(cmd []string) { b.Config.Cmd = cmd }(cmd)
	b.Config.Image = b.image

	var (
		origPath   = orig
		destPath   = dest
		remoteHash string
		isRemote   bool
		decompress = true
	)

	isRemote = utils.IsURL(orig)
	if isRemote && !allowRemote {
		return fmt.Errorf("Source can't be an URL for %s", cmdName)
	} else if utils.IsURL(orig) {
		// Initiate the download
		resp, err := utils.Download(orig)
		if err != nil {
			return err
		}

		// Create a tmp dir
		tmpDirName, err := ioutil.TempDir(b.contextPath, "docker-remote")
		if err != nil {
			return err
		}

		// Create a tmp file within our tmp dir
		tmpFileName := path.Join(tmpDirName, "tmp")
		tmpFile, err := os.OpenFile(tmpFileName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDirName)

		// Download and dump result to tmp file
		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			tmpFile.Close()
			return err
		}
		tmpFile.Close()

		// Remove the mtime of the newly created tmp file
		if err := system.UtimesNano(tmpFileName, make([]syscall.Timespec, 2)); err != nil {
			return err
		}

		origPath = path.Join(filepath.Base(tmpDirName), filepath.Base(tmpFileName))

		// Process the checksum
		r, err := archive.Tar(tmpFileName, archive.Uncompressed)
		if err != nil {
			return err
		}
		tarSum := &tarsum.TarSum{Reader: r, DisableCompression: true}
		if _, err := io.Copy(ioutil.Discard, tarSum); err != nil {
			return err
		}
		remoteHash = tarSum.Sum(nil)
		r.Close()

		// If the destination is a directory, figure out the filename.
		if strings.HasSuffix(dest, "/") {
			u, err := url.Parse(orig)
			if err != nil {
				return err
			}
			path := u.Path
			if strings.HasSuffix(path, "/") {
				path = path[:len(path)-1]
			}
			parts := strings.Split(path, "/")
			filename := parts[len(parts)-1]
			if filename == "" {
				return fmt.Errorf("cannot determine filename from url: %s", u)
			}
			destPath = dest + filename
		}
	}

	if err := b.checkPathForAddition(origPath); err != nil {
		return err
	}

	// Hash path and check the cache
	if b.Options.UtilizeCache {
		var (
			hash string
			sums = b.context.GetSums()
		)

		if remoteHash != "" {
			hash = remoteHash
		} else if fi, err := os.Stat(path.Join(b.contextPath, origPath)); err != nil {
			return err
		} else if fi.IsDir() {
			var subfiles []string
			for file, sum := range sums {
				absFile := path.Join(b.contextPath, file)
				absOrigPath := path.Join(b.contextPath, origPath)
				if strings.HasPrefix(absFile, absOrigPath) {
					subfiles = append(subfiles, sum)
				}
			}
			sort.Strings(subfiles)
			hasher := sha256.New()
			hasher.Write([]byte(strings.Join(subfiles, ",")))
			hash = "dir:" + hex.EncodeToString(hasher.Sum(nil))
		} else {
			if origPath[0] == '/' && len(origPath) > 1 {
				origPath = origPath[1:]
			}
			origPath = strings.TrimPrefix(origPath, "./")
			if h, ok := sums[origPath]; ok {
				hash = "file:" + h
			}
		}
		b.Config.Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("#(nop) %s %s in %s", cmdName, hash, dest)}
		hit, err := b.probeCache()
		if err != nil {
			return err
		}
		// If we do not have a hash, never use the cache
		if hit && hash != "" {
			return nil
		}
	}

	// Create the container
	container, _, err := b.Options.Daemon.Create(b.Config, "")
	if err != nil {
		return err
	}
	b.TmpContainers[container.ID] = struct{}{}

	if err := container.Mount(); err != nil {
		return err
	}
	defer container.Unmount()

	if !allowDecompression || isRemote {
		decompress = false
	}
	if err := b.addContext(container, origPath, destPath, decompress); err != nil {
		return err
	}

	if err := b.commit(container.ID, cmd, fmt.Sprintf("%s %s in %s", cmdName, orig, dest)); err != nil {
		return err
	}
	return nil
}

func (b *BuildFile) pullImage(name string) (*imagepkg.Image, error) {
	remote, tag := parsers.ParseRepositoryTag(name)
	pullRegistryAuth := b.Options.AuthConfig
	if len(b.Options.AuthConfigFile.Configs) > 0 {
		// The request came with a full auth config file, we prefer to use that
		endpoint, _, err := registry.ResolveRepositoryName(remote)
		if err != nil {
			return nil, err
		}
		resolvedAuth := b.Options.AuthConfigFile.ResolveAuthConfig(endpoint)
		pullRegistryAuth = &resolvedAuth
	}
	job := b.Options.Engine.Job("pull", remote, tag)
	job.SetenvBool("json", b.Options.StreamFormatter.Json())
	job.SetenvBool("parallel", true)
	job.SetenvJson("authConfig", pullRegistryAuth)
	job.Stdout.Add(b.Options.OutOld)
	if err := job.Run(); err != nil {
		return nil, err
	}
	image, err := b.Options.Daemon.Repositories().LookupImage(name)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func (b *BuildFile) processImageFrom(img *imagepkg.Image) error {
	b.image = img.ID

	if img.Config != nil {
		b.Config = img.Config
	}

	if b.Config.Env == nil || len(b.Config.Env) == 0 {
		b.Config.Env = append(b.Config.Env, "PATH="+daemon.DefaultPathEnv)
	}

	// Process ONBUILD triggers if they exist
	if nTriggers := len(b.Config.OnBuild); nTriggers != 0 {
		fmt.Fprintf(b.Options.ErrStream, "# Executing %d build triggers\n", nTriggers)
	}

	// Copy the ONBUILD triggers, and remove them from the config, since the config will be commited.
	onBuildTriggers := b.Config.OnBuild
	b.Config.OnBuild = []string{}

	// FIXME rewrite this so that builder/parser is used; right now steps in
	// onbuild are muted because we have no good way to represent the step
	// number
	for _, step := range onBuildTriggers {
		splitStep := strings.Split(step, " ")
		stepInstruction := strings.ToUpper(strings.Trim(splitStep[0], " "))
		switch stepInstruction {
		case "ONBUILD":
			return fmt.Errorf("Source image contains forbidden chained `ONBUILD ONBUILD` trigger: %s", step)
		case "MAINTAINER", "FROM":
			return fmt.Errorf("Source image contains forbidden %s trigger: %s", stepInstruction, step)
		}

		// FIXME we have to run the evaluator manually here. This does not belong
		// in this function.

		if f, ok := evaluateTable[strings.ToLower(stepInstruction)]; ok {
			if err := f(b, splitStep[1:], nil); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%s doesn't appear to be a valid Dockerfile instruction", splitStep[0])
		}
	}

	return nil
}

// probeCache checks to see if image-caching is enabled (`b.Options.UtilizeCache`)
// and if so attempts to look up the current `b.image` and `b.Config` pair
// in the current server `b.Options.Daemon`. If an image is found, probeCache returns
// `(true, nil)`. If no image is found, it returns `(false, nil)`. If there
// is any error, it returns `(false, err)`.
func (b *BuildFile) probeCache() (bool, error) {
	if b.Options.UtilizeCache {
		if cache, err := b.Options.Daemon.ImageGetCached(b.image, b.Config); err != nil {
			return false, err
		} else if cache != nil {
			fmt.Fprintf(b.Options.OutStream, " ---> Using cache\n")
			log.Debugf("[BUILDER] Use cached version")
			b.image = cache.ID
			return true, nil
		} else {
			log.Debugf("[BUILDER] Cache miss")
		}
	}
	return false, nil
}

func (b *BuildFile) create() (*daemon.Container, error) {
	if b.image == "" {
		return nil, fmt.Errorf("Please provide a source image with `from` prior to run")
	}
	b.Config.Image = b.image

	// Create the container
	c, _, err := b.Options.Daemon.Create(b.Config, "")
	if err != nil {
		return nil, err
	}
	b.TmpContainers[c.ID] = struct{}{}
	fmt.Fprintf(b.Options.OutStream, " ---> Running in %s\n", utils.TruncateID(c.ID))

	// override the entry point that may have been picked up from the base image
	c.Path = b.Config.Cmd[0]
	c.Args = b.Config.Cmd[1:]

	return c, nil
}

func (b *BuildFile) run(c *daemon.Container) error {
	var errCh chan error
	if b.Options.Verbose {
		errCh = utils.Go(func() error {
			// FIXME: call the 'attach' job so that daemon.Attach can be made private
			//
			// FIXME (LK4D4): Also, maybe makes sense to call "logs" job, it is like attach
			// but without hijacking for stdin. Also, with attach there can be race
			// condition because of some output already was printed before it.
			return <-b.Options.Daemon.Attach(c, nil, nil, b.Options.OutStream, b.Options.ErrStream)
		})
	}

	//start the container
	if err := c.Start(); err != nil {
		return err
	}

	if errCh != nil {
		if err := <-errCh; err != nil {
			return err
		}
	}

	// Wait for it to finish
	if ret, _ := c.State.WaitStop(-1 * time.Second); ret != 0 {
		err := &utils.JSONError{
			Message: fmt.Sprintf("The command %v returned a non-zero code: %d", b.Config.Cmd, ret),
			Code:    ret,
		}
		return err
	}

	return nil
}

func (b *BuildFile) checkPathForAddition(orig string) error {
	origPath := path.Join(b.contextPath, orig)
	origPath, err := filepath.EvalSymlinks(origPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: no such file or directory", orig)
		}
		return err
	}
	if !strings.HasPrefix(origPath, b.contextPath) {
		return fmt.Errorf("Forbidden path outside the build context: %s (%s)", orig, origPath)
	}
	if _, err := os.Stat(origPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: no such file or directory", orig)
		}
		return err
	}
	return nil
}

func (b *BuildFile) addContext(container *daemon.Container, orig, dest string, decompress bool) error {
	var (
		err        error
		destExists = true
		origPath   = path.Join(b.contextPath, orig)
		destPath   = path.Join(container.RootfsPath(), dest)
	)

	if destPath != container.RootfsPath() {
		destPath, err = symlink.FollowSymlinkInScope(destPath, container.RootfsPath())
		if err != nil {
			return err
		}
	}

	// Preserve the trailing '/'
	if strings.HasSuffix(dest, "/") || dest == "." {
		destPath = destPath + "/"
	}

	destStat, err := os.Stat(destPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		destExists = false
	}

	fi, err := os.Stat(origPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: no such file or directory", orig)
		}
		return err
	}

	if fi.IsDir() {
		return copyAsDirectory(origPath, destPath, destExists)
	}

	// If we are adding a remote file (or we've been told not to decompress), do not try to untar it
	if decompress {
		// First try to unpack the source as an archive
		// to support the untar feature we need to clean up the path a little bit
		// because tar is very forgiving.  First we need to strip off the archive's
		// filename from the path but this is only added if it does not end in / .
		tarDest := destPath
		if strings.HasSuffix(tarDest, "/") {
			tarDest = filepath.Dir(destPath)
		}

		// try to successfully untar the orig
		if err := archive.UntarPath(origPath, tarDest); err == nil {
			return nil
		} else if err != io.EOF {
			log.Debugf("Couldn't untar %s to %s: %s", origPath, tarDest, err)
		}
	}

	if err := os.MkdirAll(path.Dir(destPath), 0755); err != nil {
		return err
	}
	if err := archive.CopyWithTar(origPath, destPath); err != nil {
		return err
	}

	resPath := destPath
	if destExists && destStat.IsDir() {
		resPath = path.Join(destPath, path.Base(origPath))
	}

	return fixPermissions(resPath, 0, 0)
}

func copyAsDirectory(source, destination string, destinationExists bool) error {
	if err := archive.CopyWithTar(source, destination); err != nil {
		return err
	}

	if destinationExists {
		files, err := ioutil.ReadDir(source)
		if err != nil {
			return err
		}

		for _, file := range files {
			if err := fixPermissions(filepath.Join(destination, file.Name()), 0, 0); err != nil {
				return err
			}
		}
		return nil
	}

	return fixPermissions(destination, 0, 0)
}

func fixPermissions(destination string, uid, gid int) error {
	return filepath.Walk(destination, func(path string, info os.FileInfo, err error) error {
		if err := os.Lchown(path, uid, gid); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	})
}

func (b *BuildFile) clearTmp(containers map[string]struct{}) {
	for c := range containers {
		tmp := b.Options.Daemon.Get(c)
		if err := b.Options.Daemon.Destroy(tmp); err != nil {
			fmt.Fprintf(b.Options.OutStream, "Error removing intermediate container %s: %s\n", utils.TruncateID(c), err.Error())
		} else {
			delete(containers, c)
			fmt.Fprintf(b.Options.OutStream, "Removing intermediate container %s\n", utils.TruncateID(c))
		}
	}
}
