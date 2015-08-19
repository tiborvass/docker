package dockerfile

// internals for handling commands. Covers many areas and a lot of
// non-contiguous functionality. Please read the comments.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api"
	"github.com/docker/docker/builder"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/docker/docker/daemon"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/chrootarchive"
	"github.com/docker/docker/pkg/httputils"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/system"
	"github.com/docker/docker/pkg/tarsum"
	"github.com/docker/docker/pkg/urlutil"
	"github.com/docker/docker/runconfig"
)

func (b *Builder) commit(id string, autoCmd *runconfig.Command, comment string) error {
	if b.disableCommit {
		return nil
	}
	if b.image == "" && !b.noBaseImage {
		return fmt.Errorf("Please provide a source image with `from` prior to commit")
	}
	b.runConfig.Image = b.image
	if id == "" {
		cmd := b.runConfig.Cmd
		if runtime.GOOS != "windows" {
			b.runConfig.Cmd = runconfig.NewCommand("/bin/sh", "-c", "#(nop) "+comment)
		} else {
			b.runConfig.Cmd = runconfig.NewCommand("cmd", "/S /C", "REM (nop) "+comment)
		}
		// TODO: weird
		defer func(cmd *runconfig.Command) { b.runConfig.Cmd = cmd }(cmd)

		hit, err := b.probeCache()
		if err != nil {
			return err
		}
		if hit {
			return nil
		}

		container, err := b.create()
		if err != nil {
			return err
		}
		id = container.ID

		if err := container.Mount(); err != nil {
			return err
		}
		defer container.Unmount()
	}

	//container, err := b.Daemon.Get(id)
	container, err := b.docker.Container(id)
	if err != nil {
		return err
	}

	// Note: Actually copy the struct
	autoConfig := *b.runConfig
	autoConfig.Cmd = autoCmd

	commitCfg := &daemon.ContainerCommitConfig{
		Author: b.maintainer,
		Pause:  true,
		Config: &autoConfig,
	}

	// Commit the container
	//image, err := b.Daemon.Commit(container, commitCfg)
	image, err := b.docker.Commit(container, commitCfg)
	if err != nil {
		return err
	}
	// TODO: transaction
	//b.Daemon.Graph().Retain(b.id, image.ID)
	//b.activeImages = append(b.activeImages, image.ID)
	b.image = image.ID
	return nil
}

type copyInfo struct {
	builder.ContextEntry
	decompress bool
}

func wildcardWalk(infos *[]copyInfo, pattern string, decompress bool) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "" {
			return nil
		}
		if match, _ := filepath.Match(pattern, info.Name()); !match {
			return nil
		}
		entry, ok := info.(builder.HashFileInfo)
		if !ok {
			return fmt.Errorf("builder: unexpected type (%T) for FileInfo in wildcardWalk", info)
		}
		*infos = append(*infos, copyInfo{entry, decompress})
		return nil
	}
}

func convertHash(fi builder.HashFileInfo) string {
	if !fi.IsDir() {
		return "file:" + fi.Hash
	}
	var subfiles []string
	//b.context.Walk()
	sort.Strings(subfiles)
	hasher := sha256.New()
	hasher.Write([]byte(strings.Join(subfiles, ",")))
	return "dir" + hex.EncodeToString(hasher.Sum(nil))
}

func containsWildcards(name string) bool {
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '\\' {
			i++
		} else if ch == '*' || ch == '?' || ch == '[' {
			return true
		}
	}
	return false
}

func (b *Builder) runContextCommand(args []string, allowRemote bool, allowLocalDecompression bool, cmdName string) error {
	if b.context == nil {
		return fmt.Errorf("No context given. Impossible to use %s", cmdName)
	}

	if len(args) < 2 {
		return fmt.Errorf("Invalid %s format - at least two arguments required", cmdName)
	}

	// Work in daemon-specific filepath semantics
	dest := filepath.FromSlash(args[len(args)-1]) // last one is always the dest

	b.runConfig.Image = b.image

	var infos []copyInfo

	// Loop through each src file and calculate the info we need to
	// do the copy (e.g. hash value if cached).  Don't actually do
	// the copy until we've looked at all src files
	var err error
	for _, orig := range args[:len(args)-1] {
		var entry builder.ContextEntry
		decompress := allowLocalDecompression
		if urlutil.IsURL(orig) {
			if !allowRemote {
				return fmt.Errorf("Source can't be a URL for %s", cmdName)
			}
			entry, err = b.download(orig)
			defer os.RemoveAll(filepath.Dir(entry.Path()))
			if err != nil {
				return err
			}
			decompress = false
		} else {
			if containsWildcards(orig) {
				if err := b.context.Walk(wildcardWalk(&infos, orig, decompress)); err != nil {
					return err
				}
				continue
			}

			// Must be a dir or a file

			if err := b.checkPathForAddition(orig); err != nil {
				return err
			}

			entry, err = b.context.Get(orig)
			if err != nil {
				return err
			}

			if entry.IsDir() {
			}
		}
		/*
			var hash string
			if !st.IsDir() {
				hash = "file:" + st.Hash
			} else {
				var subfiles []string
				b.context.Walk()
				sort.Strings(subfiles)
				hasher := sha256.New()
				hasher.Write([]byte(strings.Join(subfiles, ",")))
				hash = "dir" + hex.EncodeToString(hasher.Sum(nil))
			}
		*/
		infos = append(infos, copyInfo{entry, decompress})
	}

	if len(infos) == 0 {
		return fmt.Errorf("No source files were specified")
	}
	if len(infos) > 1 && !strings.HasSuffix(dest, string(os.PathSeparator)) {
		return fmt.Errorf("When using %s with more than one source file, the destination must be a directory and end with a /", cmdName)
	}

	// For backwards compat, if there's just one info then use it as the
	// cache look-up string, otherwise hash 'em all into one
	var srcHash string
	var origPaths string

	if len(infos) == 1 {
		fi := infos[0].ContextEntry
		origPaths = fi.Name()
		if fi, ok := fi.(builder.HashFileInfo); ok {
			if hash := convertHash(fi); len(hash) > 0 {
				srcHash = hash
			}
		}
	} else {
		var hashs []string
		var origs []string
		for _, info := range infos {
			fi := info.ContextEntry
			origs = append(origs, fi.Name())
			if fi, ok := fi.(builder.HashFileInfo); ok {
				if hash := convertHash(fi); len(hash) > 0 {
					hashs = append(hashs, hash)
				}
			}
		}
		hasher := sha256.New()
		hasher.Write([]byte(strings.Join(hashs, ",")))
		srcHash = "multi:" + hex.EncodeToString(hasher.Sum(nil))
		origPaths = strings.Join(origs, " ")
	}

	cmd := b.runConfig.Cmd
	if runtime.GOOS != "windows" {
		b.runConfig.Cmd = runconfig.NewCommand("/bin/sh", "-c", fmt.Sprintf("#(nop) %s %s in %s", cmdName, srcHash, dest))
	} else {
		b.runConfig.Cmd = runconfig.NewCommand("cmd", "/S /C", fmt.Sprintf("REM (nop) %s %s in %s", cmdName, srcHash, dest))
	}
	defer func(cmd *runconfig.Command) { b.runConfig.Cmd = cmd }(cmd)

	if hit, err := b.probeCache(); err != nil {
		return err
	} else if hit {
		return nil
	}

	//container, _, err := b.Daemon.Create(b.runConfig, nil, "")
	container, err := b.docker.Create(b.runConfig, nil)
	/*
		if err := container.Mount(); err != nil {
			return err
		}
		defer container.Unmount()
	*/
	if err != nil {
		return err
	}
	defer container.Unmount()
	b.tmpContainers[container.ID] = struct{}{}

	for _, info := range infos {
		srcPath := info.Path()

		// If we are adding a remote file (or we've been told not to decompress), do not try to untar it
		// (used for ADD)
		if info.decompress && !info.IsDir() {
			// First try to unpack the source as an archive
			// to support the untar feature we need to clean up the path a little bit
			// because tar is very forgiving.  First we need to strip off the archive's
			// filename from the path but this is only added if it does not end in slash
			tarDest := dest
			if strings.HasSuffix(tarDest, string(os.PathSeparator)) {
				tarDest = filepath.Dir(dest)
			}

			// try to successfully untar the orig
			if err := chrootarchive.UntarPath(srcPath, tarDest); err != nil {
				continue
			} else if err != io.EOF {
				logrus.Debugf("Couldn't untar to %s: %v", tarDest, err)
			}
		}
		if err := b.docker.Copy(container, dest, srcPath); err != nil {
			return err
		}
	}

	if err := b.commit(container.ID, cmd, fmt.Sprintf("%s %s in %s", cmdName, origPaths, dest)); err != nil {
		return err
	}
	return nil
}

func (b *Builder) download(srcURL string) (builder.ContextEntry, error) {
	// get filename from URL
	u, err := url.Parse(srcURL)
	if err != nil {
		return nil, err
	}
	path := u.Path
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		path = path[:len(path)-1]
	}
	parts := strings.Split(path, string(os.PathSeparator))
	filename := parts[len(parts)-1]
	if filename == "" {
		return nil, fmt.Errorf("cannot determine filename from url: %s", u)
	}

	// Initiate the download
	resp, err := httputils.Download(srcURL)
	if err != nil {
		return nil, err
	}

	// Prepare file in a tmp dir
	tmpDir, err := ioutil.TempDir("", "docker-remote")
	if err != nil {
		return nil, err
	}
	tmpFileName := filepath.Join(tmpDir, filename)
	tmpFile, err := os.OpenFile(tmpFileName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}

	// Download and dump result to tmp file
	// TODO: put a better progressreader back
	/*
		if _, err := io.Copy(tmpFile, progressreader.New(progressreader.Config{
			In:       resp.Body,
			Out:      b.Stdout,
			Formatter: ...,
			Size:     resp.ContentLength,
			NewLines: true,
			ID:       "",
			Action:   "Downloading",
	*/
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return nil, err
	}
	fmt.Fprintln(b.Stdout)
	// ignoring error because the file was already opened successfully
	tmpFileSt, _ := tmpFile.Stat()
	tmpFile.Close()

	// Set the mtime to the Last-Modified header value if present
	// Otherwise just remove atime and mtime
	times := make([]syscall.Timespec, 2)

	lastMod := resp.Header.Get("Last-Modified")
	if lastMod != "" {
		mTime, err := http.ParseTime(lastMod)
		// If we can't parse it then just let it default to 'zero'
		// otherwise use the parsed time value
		if err == nil {
			times[1] = syscall.NsecToTimespec(mTime.UnixNano())
		}
	}

	if err := system.UtimesNano(tmpFileName, times); err != nil {
		return nil, err
	}

	//ci.origPath = filepath.Join(filepath.Base(tmpDirName), filepath.Base(tmpFileName))

	// If the destination is a directory, figure out the filename.
	//if strings.HasSuffix(ci.destPath, string(os.PathSeparator)) {

	//ci.destPath = ci.destPath + filename
	//}

	// Calc the checksum, even if we're using the cache
	r, err := archive.Tar(tmpFileName, archive.Uncompressed)
	if err != nil {
		return nil, err
	}
	tarSum, err := tarsum.NewTarSum(r, true, tarsum.Version1)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(ioutil.Discard, tarSum); err != nil {
		return nil, err
	}
	hash := tarSum.Sum(nil)
	r.Close()

	return builder.HashFileInfo{builder.FileInfo{tmpFileSt, tmpFileName}, hash}, nil
}

/*
func (b *Builder) calcCopyInfo(cmdName string, cInfos []*copyInfo, origPath string, destPath string, allowRemote bool, allowDecompression bool, allowWildcards bool) ([]*copyInfo, error) {

	// Work in daemon-specific OS filepath semantics. However, we save
	// the the origPath passed in here, as it might also be a URL which
	// we need to check for in this function.
	//TODO: not used: passedInOrigPath := origPath
	origPath = filepath.FromSlash(origPath)
	destPath = filepath.FromSlash(destPath)

	if origPath != "" && origPath[0] == os.PathSeparator && len(origPath) > 1 {
		origPath = origPath[1:]
	}
	origPath = strings.TrimPrefix(origPath, "."+string(os.PathSeparator))

	// Twiddle the destPath when its a relative path - meaning, make it
	// relative to the WORKINGDIR
	if !filepath.IsAbs(destPath) {
		hasSlash := strings.HasSuffix(destPath, string(os.PathSeparator))
		destPath = filepath.Join(string(os.PathSeparator), filepath.FromSlash(b.runConfig.WorkingDir), destPath)

		// Make sure we preserve any trailing slash
		if hasSlash {
			destPath += string(os.PathSeparator)
		}
	}

	ci := &copyInfo{origPath: origPath, destPath: destPath}
	ctx, ok := b.context.(builder.ContextWithCache)
	if ok {
		h, err := ctx.Hash(origPath)
		if err != nil {
			return nil, err
		}
		ci.hash = h
	}

	return append(cInfos, ci), nil

	/*
		// TODO: Deal with wildcards
		if allowWildcards && containsWildcards(origPath) {
			for _, fileInfo := range b.context.GetSums() {
				if fileInfo.Name() == "" {
					continue
				}
				match, _ := filepath.Match(origPath, fileInfo.Name())
				if !match {
					continue
				}

				// Note we set allowWildcards to false in case the name has
				// a * in it
				calcCopyInfo(b, cmdName, cInfos, fileInfo.Name(), destPath, allowRemote, allowDecompression, false)
			}
			return nil
		}

		// Must be a dir or a file

		if err := b.checkPathForAddition(origPath); err != nil {
			return err
		}
		fi, _ := os.Stat(filepath.Join(b.contextPath, origPath))

		ci := copyInfo{}
		ci.origPath = origPath
		ci.hash = origPath
		ci.destPath = destPath
		ci.decompress = allowDecompression
		*cInfos = append(*cInfos, &ci)

		// Deal with the single file case
		if !fi.IsDir() {
			// This will match first file in sums of the archive
			fis := b.context.GetSums().GetFile(ci.origPath)
			if fis != nil {
				ci.hash = "file:" + fis.Sum()
			}
			return nil
		}

		// Must be a dir
		var subfiles []string
		absOrigPath := filepath.Join(b.contextPath, ci.origPath)

		// Add a trailing / to make sure we only pick up nested files under
		// the dir and not sibling files of the dir that just happen to
		// start with the same chars
		if !strings.HasSuffix(absOrigPath, string(os.PathSeparator)) {
			absOrigPath += string(os.PathSeparator)
		}

		// Need path w/o slash too to find matching dir w/o trailing slash
		absOrigPathNoSlash := absOrigPath[:len(absOrigPath)-1]

		for _, fileInfo := range b.context.GetSums() {
			absFile := filepath.Join(b.contextPath, fileInfo.Name())
			// Any file in the context that starts with the given path will be
			// picked up and its hashcode used.  However, we'll exclude the
			// root dir itself.  We do this for a coupel of reasons:
			// 1 - ADD/COPY will not copy the dir itself, just its children
			//     so there's no reason to include it in the hash calc
			// 2 - the metadata on the dir will change when any child file
			//     changes.  This will lead to a miss in the cache check if that
			//     child file is in the .dockerignore list.
			if strings.HasPrefix(absFile, absOrigPath) && absFile != absOrigPathNoSlash {
				subfiles = append(subfiles, fileInfo.Sum())
			}
		}
		sort.Strings(subfiles)
		hasher := sha256.New()
		hasher.Write([]byte(strings.Join(subfiles, ",")))
		ci.hash = "dir:" + hex.EncodeToString(hasher.Sum(nil))

		return nil
}
*/

func (b *Builder) processImageFrom(img *image.Image) error {
	b.image = img.ID

	if img.Config != nil {
		b.runConfig = img.Config
	}

	// The default path will be blank on Windows (set by HCS)
	if len(b.runConfig.Env) == 0 && daemon.DefaultPathEnv != "" {
		b.runConfig.Env = append(b.runConfig.Env, "PATH="+daemon.DefaultPathEnv)
	}

	// Process ONBUILD triggers if they exist
	if nTriggers := len(b.runConfig.OnBuild); nTriggers != 0 {
		fmt.Fprintf(b.Stderr, "# Executing %d build triggers\n", nTriggers)
	}

	// Copy the ONBUILD triggers, and remove them from the config, since the config will be committed.
	onBuildTriggers := b.runConfig.OnBuild
	b.runConfig.OnBuild = []string{}

	// parse the ONBUILD triggers by invoking the parser
	for stepN, step := range onBuildTriggers {
		ast, err := parser.Parse(strings.NewReader(step))
		if err != nil {
			return err
		}

		for i, n := range ast.Children {
			switch strings.ToUpper(n.Value) {
			case "ONBUILD":
				return fmt.Errorf("Chaining ONBUILD via `ONBUILD ONBUILD` isn't allowed")
			case "MAINTAINER", "FROM":
				return fmt.Errorf("%s isn't allowed as an ONBUILD trigger", n.Value)
			}

			fmt.Fprintf(b.Stdout, "Trigger %d, %s\n", stepN, step)

			if err := b.dispatch(i, n); err != nil {
				return err
			}
		}
	}

	return nil
}

// probeCache checks to see if image-caching is enabled (`b.UseCache`)
// and if so attempts to look up the current `b.image` and `b.runConfig` pair
// in the current server `b.Daemon`. If an image is found, probeCache returns
// `(true, nil)`. If no image is found, it returns `(false, nil)`. If there
// is any error, it returns `(false, err)`.
func (b *Builder) probeCache() (bool, error) {
	c, ok := b.docker.(builder.ImageCache)
	if !ok || !b.UseCache || b.cacheBusted {
		return false, nil
	}
	cache, err := c.GetCachedImage(builder.ImageID(b.image), b.runConfig)
	if err != nil {
		return false, err
	}
	if len(cache) == 0 {
		logrus.Debugf("[BUILDER] Cache miss")
		b.cacheBusted = true
		return false, nil
	}

	fmt.Fprintf(b.Stdout, " ---> Using cache\n")
	logrus.Debugf("[BUILDER] Use cached version")
	b.image = string(cache)
	// TODO: transaction
	//b.Daemon.Graph().Retain(b.id, cache.ID)
	//b.activeImages = append(b.activeImages, cache.ID)
	return true, nil
}

func (b *Builder) create() (*daemon.Container, error) {
	if b.image == "" && !b.noBaseImage {
		return nil, fmt.Errorf("Please provide a source image with `from` prior to run")
	}
	b.runConfig.Image = b.image

	// TODO: why not embed a hostconfig in builder?
	hostConfig := &runconfig.HostConfig{
		CPUShares:    b.CPUShares,
		CPUPeriod:    b.CPUPeriod,
		CPUQuota:     b.CPUQuota,
		CpusetCpus:   b.CPUSetCpus,
		CpusetMems:   b.CPUSetMems,
		CgroupParent: b.CgroupParent,
		Memory:       b.Memory,
		MemorySwap:   b.MemorySwap,
		Ulimits:      b.Ulimits,
	}

	config := *b.runConfig

	// Create the container
	c, err := b.docker.Create(b.runConfig, hostConfig)
	if err != nil {
		return nil, err
	}
	defer c.Unmount()
	// TODO: get warnings from err returned by Create
	//for _, warning := range warnings {
	//	fmt.Fprintf(b.Stdout, " ---> [Warning] %s\n", warning)
	//}

	b.tmpContainers[c.ID] = struct{}{}
	fmt.Fprintf(b.Stdout, " ---> Running in %s\n", stringid.TruncateID(c.ID))

	if config.Cmd.Len() > 0 {
		// override the entry point that may have been picked up from the base image
		s := config.Cmd.Slice()
		c.Path = s[0]
		c.Args = s[1:]
	} else {
		config.Cmd = runconfig.NewCommand()
	}

	return c, nil
}

func (b *Builder) run(c *daemon.Container) error {
	var errCh chan error
	if b.Verbose {
		errCh = c.Attach(nil, b.Stdout, b.Stderr)
	}

	//start the container
	if err := c.Start(); err != nil {
		return err
	}

	// TODO: cancel
	/*
		finished := make(chan struct{})
		defer close(finished)
		go func() {
			select {
			case <-b.cancelled:
				logrus.Debugln("Build cancelled, killing container:", c.ID)
				c.Kill()
			case <-finished:
			}
		}()
	*/

	if b.Verbose {
		// Block on reading output from container, stop on err or chan closed
		if err := <-errCh; err != nil {
			return err
		}
	}

	// Wait for it to finish
	if ret, _ := c.WaitStop(-1 * time.Second); ret != 0 {
		return &jsonmessage.JSONError{
			Message: fmt.Sprintf("The command '%s' returned a non-zero code: %d", b.runConfig.Cmd.ToString(), ret),
			Code:    ret,
		}
	}

	return nil
}

func (b *Builder) checkPathForAddition(orig string) error {
	return nil
	// TODO: move to builder/tarsum ?
	/*
		origPath := filepath.Join(b.contextPath, orig)
		origPath, err := filepath.EvalSymlinks(origPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s: no such file or directory", orig)
			}
			return err
		}
		contextPath, err := filepath.EvalSymlinks(b.contextPath)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(origPath, contextPath) {
			return fmt.Errorf("Forbidden path outside the build context: %s (%s)", orig, origPath)
		}
		if _, err := os.Stat(origPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s: no such file or directory", orig)
			}
			return err
		}
		return nil
	*/
}

/*
func copyAsDirectory(source, destination string, destExisted bool) error {
	if err := chrootarchive.CopyWithTar(source, destination); err != nil {
		return err
	}
	return fixPermissions(source, destination, 0, 0, destExisted)
}

*/
func (b *Builder) clearTmp() {
	for c := range b.tmpContainers {
		rmConfig := &daemon.ContainerRmConfig{
			ForceRemove:  true,
			RemoveVolume: true,
		}
		//if err := b.Daemon.ContainerRm(c, rmConfig); err != nil {
		if err := b.docker.Remove(c, rmConfig); err != nil {
			fmt.Fprintf(b.Stdout, "Error removing intermediate container %s: %v\n", stringid.TruncateID(c), err)
			return
		}
		delete(b.tmpContainers, c)
		fmt.Fprintf(b.Stdout, "Removing intermediate container %s\n", stringid.TruncateID(c))
	}
}

// Reads a Dockerfile from the current context. It assumes that the
// 'filename' is a relative path from the root of the context
func (b *Builder) readDockerfile() error {
	// If no -f was specified then look for 'Dockerfile'. If we can't find
	// that then look for 'dockerfile'.  If neither are found then default
	// back to 'Dockerfile' and use that in the error message.
	if b.DockerfileName == "" {
		b.DockerfileName = api.DefaultDockerfileName

		if _, err := b.context.Get(b.DockerfileName); os.IsNotExist(err) {
			lowercase := strings.ToLower(b.DockerfileName)
			if _, err := b.context.Get(lowercase); err == nil {
				b.DockerfileName = lowercase
			}
		}
	}

	f, err := b.context.Open(b.DockerfileName)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Cannot locate specified Dockerfile: %s", b.DockerfileName)
		}
		return err
	}
	if f, ok := f.(*os.File); ok {
		// ignoring error because Open already succeeded
		fi, _ := f.Stat()
		if fi.Size() == 0 {
			return fmt.Errorf("%s cannot be empty", b.DockerfileName)
		}
	}

	b.dockerfile, err = parser.Parse(f)
	f.Close()
	if err != nil {
		return err
	}

	// After the Dockerfile has been parsed, we need to check the .dockerignore
	// file for either "Dockerfile" or ".dockerignore", and if either are
	// present then erase them from the build context. These files should never
	// have been sent from the client but we did send them to make sure that
	// we had the Dockerfile to actually parse, and then we also need the
	// .dockerignore file to know whether either file should be removed.
	// Note that this assumes the Dockerfile has been read into memory and
	// is now safe to be removed.

	// TODO: should this be handled as an implementation of Context?

	/*
		dockerignore, err := context.Open(".dockerignore")
		excludes, _ := utils.ReadDockerIgnore(dockerignore)
		if rm, _ := fileutils.Matches(".dockerignore", excludes); rm == true {
			os.Remove(filepath.Join(b.contextPath, ".dockerignore"))
			b.context.(tarsum.BuilderContext).Remove(".dockerignore")
		}
		if rm, _ := fileutils.Matches(b.dockerfileName, excludes); rm == true {
			os.Remove(filepath.Join(b.contextPath, b.dockerfileName))
			b.context.(tarsum.BuilderContext).Remove(b.dockerfileName)
		}
	*/

	return nil
}
