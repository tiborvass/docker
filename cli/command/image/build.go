package image

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/builder"
	"github.com/tiborvass/docker/builder/dockerignore"
	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cli/command"
	"github.com/tiborvass/docker/opts"
	"github.com/tiborvass/docker/pkg/archive"
	"github.com/tiborvass/docker/pkg/fileutils"
	"github.com/tiborvass/docker/pkg/jsonmessage"
	"github.com/tiborvass/docker/pkg/progress"
	"github.com/tiborvass/docker/pkg/streamformatter"
	"github.com/tiborvass/docker/pkg/urlutil"
	"github.com/tiborvass/docker/reference"
	runconfigopts "github.com/tiborvass/docker/runconfig/opts"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type buildOptions struct {
	context        string
	dockerfileName string
	tags           opts.ListOpts
	labels         []string
	buildArgs      opts.ListOpts
	ulimits        *runconfigopts.UlimitOpt
	memory         string
	memorySwap     string
	shmSize        string
	cpuShares      int64
	cpuPeriod      int64
	cpuQuota       int64
	cpuSetCpus     string
	cpuSetMems     string
	cgroupParent   string
	isolation      string
	quiet          bool
	noCache        bool
	rm             bool
	forceRm        bool
	pull           bool
}

// NewBuildCommand creates a new `docker build` command
func NewBuildCommand(dockerCli *command.DockerCli) *cobra.Command {
	ulimits := make(map[string]*units.Ulimit)
	options := buildOptions{
		tags:      opts.NewListOpts(validateTag),
		buildArgs: opts.NewListOpts(runconfigopts.ValidateEnv),
		ulimits:   runconfigopts.NewUlimitOpt(&ulimits),
	}

	cmd := &cobra.Command{
		Use:   "build [OPTIONS] PATH | URL | -",
		Short: "Build an image from a Dockerfile",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.context = args[0]
			return runBuild(dockerCli, options)
		},
	}

	flags := cmd.Flags()

	flags.VarP(&options.tags, "tag", "t", "Name and optionally a tag in the 'name:tag' format")
	flags.Var(&options.buildArgs, "build-arg", "Set build-time variables")
	flags.Var(options.ulimits, "ulimit", "Ulimit options")
	flags.StringVarP(&options.dockerfileName, "file", "f", "", "Name of the Dockerfile (Default is 'PATH/Dockerfile')")
	flags.StringVarP(&options.memory, "memory", "m", "", "Memory limit")
	flags.StringVar(&options.memorySwap, "memory-swap", "", "Swap limit equal to memory plus swap: '-1' to enable unlimited swap")
	flags.StringVar(&options.shmSize, "shm-size", "", "Size of /dev/shm, default value is 64MB")
	flags.Int64VarP(&options.cpuShares, "cpu-shares", "c", 0, "CPU shares (relative weight)")
	flags.Int64Var(&options.cpuPeriod, "cpu-period", 0, "Limit the CPU CFS (Completely Fair Scheduler) period")
	flags.Int64Var(&options.cpuQuota, "cpu-quota", 0, "Limit the CPU CFS (Completely Fair Scheduler) quota")
	flags.StringVar(&options.cpuSetCpus, "cpuset-cpus", "", "CPUs in which to allow execution (0-3, 0,1)")
	flags.StringVar(&options.cpuSetMems, "cpuset-mems", "", "MEMs in which to allow execution (0-3, 0,1)")
	flags.StringVar(&options.cgroupParent, "cgroup-parent", "", "Optional parent cgroup for the container")
	flags.StringVar(&options.isolation, "isolation", "", "Container isolation technology")
	flags.StringSliceVar(&options.labels, "label", []string{}, "Set metadata for an image")
	flags.BoolVar(&options.noCache, "no-cache", false, "Do not use cache when building the image")
	flags.BoolVar(&options.rm, "rm", true, "Remove intermediate containers after a successful build")
	flags.BoolVar(&options.forceRm, "force-rm", false, "Always remove intermediate containers")
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Suppress the build output and print image ID on success")
	flags.BoolVar(&options.pull, "pull", false, "Always attempt to pull a newer version of the image")

	command.AddTrustedFlags(flags, true)

	return cmd
}

// lastProgressOutput is the same as progress.Output except
// that it only output with the last update. It is used in
// non terminal scenarios to depresss verbose messages
type lastProgressOutput struct {
	output progress.Output
}

// WriteProgress formats progress information from a ProgressReader.
func (out *lastProgressOutput) WriteProgress(prog progress.Progress) error {
	if !prog.LastUpdate {
		return nil
	}

	return out.output.WriteProgress(prog)
}

func runBuild(dockerCli *command.DockerCli, options buildOptions) error {

	var (
		buildCtx io.ReadCloser
		err      error
	)

	specifiedContext := options.context

	var (
		contextDir    string
		tempDir       string
		relDockerfile string
		progBuff      io.Writer
		buildBuff     io.Writer
	)

	progBuff = dockerCli.Out()
	buildBuff = dockerCli.Out()
	if options.quiet {
		progBuff = bytes.NewBuffer(nil)
		buildBuff = bytes.NewBuffer(nil)
	}

	switch {
	case specifiedContext == "-":
		buildCtx, relDockerfile, err = builder.GetContextFromReader(dockerCli.In(), options.dockerfileName)
	case urlutil.IsGitURL(specifiedContext):
		tempDir, relDockerfile, err = builder.GetContextFromGitURL(specifiedContext, options.dockerfileName)
	case urlutil.IsURL(specifiedContext):
		buildCtx, relDockerfile, err = builder.GetContextFromURL(progBuff, specifiedContext, options.dockerfileName)
	default:
		contextDir, relDockerfile, err = builder.GetContextFromLocalDir(specifiedContext, options.dockerfileName)
	}

	if err != nil {
		if options.quiet && urlutil.IsURL(specifiedContext) {
			fmt.Fprintln(dockerCli.Err(), progBuff)
		}
		return fmt.Errorf("unable to prepare context: %s", err)
	}

	if tempDir != "" {
		defer os.RemoveAll(tempDir)
		contextDir = tempDir
	}

	if buildCtx == nil {
		// And canonicalize dockerfile name to a platform-independent one
		relDockerfile, err = archive.CanonicalTarNameForPath(relDockerfile)
		if err != nil {
			return fmt.Errorf("cannot canonicalize dockerfile path %s: %v", relDockerfile, err)
		}

		f, err := os.Open(filepath.Join(contextDir, ".dockerignore"))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		defer f.Close()

		var excludes []string
		if err == nil {
			excludes, err = dockerignore.ReadAll(f)
			if err != nil {
				return err
			}
		}

		if err := builder.ValidateContextDirectory(contextDir, excludes); err != nil {
			return fmt.Errorf("Error checking context: '%s'.", err)
		}

		// If .dockerignore mentions .dockerignore or the Dockerfile
		// then make sure we send both files over to the daemon
		// because Dockerfile is, obviously, needed no matter what, and
		// .dockerignore is needed to know if either one needs to be
		// removed. The daemon will remove them for us, if needed, after it
		// parses the Dockerfile. Ignore errors here, as they will have been
		// caught by validateContextDirectory above.
		var includes = []string{"."}
		keepThem1, _ := fileutils.Matches(".dockerignore", excludes)
		keepThem2, _ := fileutils.Matches(relDockerfile, excludes)
		if keepThem1 || keepThem2 {
			includes = append(includes, ".dockerignore", relDockerfile)
		}

		buildCtx, err = archive.TarWithOptions(contextDir, &archive.TarOptions{
			Compression:     archive.Uncompressed,
			ExcludePatterns: excludes,
			IncludeFiles:    includes,
		})
		if err != nil {
			return err
		}
	}

	ctx := context.Background()

	var resolvedTags []*resolvedTag
	if command.IsTrusted() {
		translator := func(ctx context.Context, ref reference.NamedTagged) (reference.Canonical, error) {
			return TrustedReference(ctx, dockerCli, ref)
		}
		// Wrap the tar archive to replace the Dockerfile entry with the rewritten
		// Dockerfile which uses trusted pulls.
		buildCtx = replaceDockerfileTarWrapper(ctx, buildCtx, relDockerfile, translator, &resolvedTags)
	}

	// Setup an upload progress bar
	progressOutput := streamformatter.NewStreamFormatter().NewProgressOutput(progBuff, true)
	if !dockerCli.Out().IsTerminal() {
		progressOutput = &lastProgressOutput{output: progressOutput}
	}

	var body io.Reader = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")

	var memory int64
	if options.memory != "" {
		parsedMemory, err := units.RAMInBytes(options.memory)
		if err != nil {
			return err
		}
		memory = parsedMemory
	}

	var memorySwap int64
	if options.memorySwap != "" {
		if options.memorySwap == "-1" {
			memorySwap = -1
		} else {
			parsedMemorySwap, err := units.RAMInBytes(options.memorySwap)
			if err != nil {
				return err
			}
			memorySwap = parsedMemorySwap
		}
	}

	var shmSize int64
	if options.shmSize != "" {
		shmSize, err = units.RAMInBytes(options.shmSize)
		if err != nil {
			return err
		}
	}

	authConfig, _ := dockerCli.CredentialsStore().GetAll()
	buildOptions := types.ImageBuildOptions{
		Memory:         memory,
		MemorySwap:     memorySwap,
		Tags:           options.tags.GetAll(),
		SuppressOutput: options.quiet,
		NoCache:        options.noCache,
		Remove:         options.rm,
		ForceRemove:    options.forceRm,
		PullParent:     options.pull,
		Isolation:      container.Isolation(options.isolation),
		CPUSetCPUs:     options.cpuSetCpus,
		CPUSetMems:     options.cpuSetMems,
		CPUShares:      options.cpuShares,
		CPUQuota:       options.cpuQuota,
		CPUPeriod:      options.cpuPeriod,
		CgroupParent:   options.cgroupParent,
		Dockerfile:     relDockerfile,
		ShmSize:        shmSize,
		Ulimits:        options.ulimits.GetList(),
		BuildArgs:      runconfigopts.ConvertKVStringsToMap(options.buildArgs.GetAll()),
		AuthConfigs:    authConfig,
		Labels:         runconfigopts.ConvertKVStringsToMap(options.labels),
	}

	response, err := dockerCli.Client().ImageBuild(ctx, body, buildOptions)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	err = jsonmessage.DisplayJSONMessagesStream(response.Body, buildBuff, dockerCli.Out().FD(), dockerCli.Out().IsTerminal(), nil)
	if err != nil {
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			// If no error code is set, default to 1
			if jerr.Code == 0 {
				jerr.Code = 1
			}
			if options.quiet {
				fmt.Fprintf(dockerCli.Err(), "%s%s", progBuff, buildBuff)
			}
			return cli.StatusError{Status: jerr.Message, StatusCode: jerr.Code}
		}
	}

	// Windows: show error message about modified file permissions if the
	// daemon isn't running Windows.
	if response.OSType != "windows" && runtime.GOOS == "windows" && !options.quiet {
		fmt.Fprintln(dockerCli.Err(), `SECURITY WARNING: You are building a Docker image from Windows against a non-Windows Docker host. All files and directories added to build context will have '-rwxr-xr-x' permissions. It is recommended to double check and reset permissions for sensitive files and directories.`)
	}

	// Everything worked so if -q was provided the output from the daemon
	// should be just the image ID and we'll print that to stdout.
	if options.quiet {
		fmt.Fprintf(dockerCli.Out(), "%s", buildBuff)
	}

	if command.IsTrusted() {
		// Since the build was successful, now we must tag any of the resolved
		// images from the above Dockerfile rewrite.
		for _, resolved := range resolvedTags {
			if err := TagTrusted(ctx, dockerCli, resolved.digestRef, resolved.tagRef); err != nil {
				return err
			}
		}
	}

	return nil
}

type translatorFunc func(context.Context, reference.NamedTagged) (reference.Canonical, error)

// validateTag checks if the given image name can be resolved.
func validateTag(rawRepo string) (string, error) {
	_, err := reference.ParseNamed(rawRepo)
	if err != nil {
		return "", err
	}

	return rawRepo, nil
}

var dockerfileFromLinePattern = regexp.MustCompile(`(?i)^[\s]*FROM[ \f\r\t\v]+(?P<image>[^ \f\r\t\v\n#]+)`)

// resolvedTag records the repository, tag, and resolved digest reference
// from a Dockerfile rewrite.
type resolvedTag struct {
	digestRef reference.Canonical
	tagRef    reference.NamedTagged
}

// rewriteDockerfileFrom rewrites the given Dockerfile by resolving images in
// "FROM <image>" instructions to a digest reference. `translator` is a
// function that takes a repository name and tag reference and returns a
// trusted digest reference.
func rewriteDockerfileFrom(ctx context.Context, dockerfile io.Reader, translator translatorFunc) (newDockerfile []byte, resolvedTags []*resolvedTag, err error) {
	scanner := bufio.NewScanner(dockerfile)
	buf := bytes.NewBuffer(nil)

	// Scan the lines of the Dockerfile, looking for a "FROM" line.
	for scanner.Scan() {
		line := scanner.Text()

		matches := dockerfileFromLinePattern.FindStringSubmatch(line)
		if matches != nil && matches[1] != api.NoBaseImageSpecifier {
			// Replace the line with a resolved "FROM repo@digest"
			ref, err := reference.ParseNamed(matches[1])
			if err != nil {
				return nil, nil, err
			}
			ref = reference.WithDefaultTag(ref)
			if ref, ok := ref.(reference.NamedTagged); ok && command.IsTrusted() {
				trustedRef, err := translator(ctx, ref)
				if err != nil {
					return nil, nil, err
				}

				line = dockerfileFromLinePattern.ReplaceAllLiteralString(line, fmt.Sprintf("FROM %s", trustedRef.String()))
				resolvedTags = append(resolvedTags, &resolvedTag{
					digestRef: trustedRef,
					tagRef:    ref,
				})
			}
		}

		_, err := fmt.Fprintln(buf, line)
		if err != nil {
			return nil, nil, err
		}
	}

	return buf.Bytes(), resolvedTags, scanner.Err()
}

// replaceDockerfileTarWrapper wraps the given input tar archive stream and
// replaces the entry with the given Dockerfile name with the contents of the
// new Dockerfile. Returns a new tar archive stream with the replaced
// Dockerfile.
func replaceDockerfileTarWrapper(ctx context.Context, inputTarStream io.ReadCloser, dockerfileName string, translator translatorFunc, resolvedTags *[]*resolvedTag) io.ReadCloser {
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		tarReader := tar.NewReader(inputTarStream)
		tarWriter := tar.NewWriter(pipeWriter)

		defer inputTarStream.Close()

		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				// Signals end of archive.
				tarWriter.Close()
				pipeWriter.Close()
				return
			}
			if err != nil {
				pipeWriter.CloseWithError(err)
				return
			}

			content := io.Reader(tarReader)
			if hdr.Name == dockerfileName {
				// This entry is the Dockerfile. Since the tar archive was
				// generated from a directory on the local filesystem, the
				// Dockerfile will only appear once in the archive.
				var newDockerfile []byte
				newDockerfile, *resolvedTags, err = rewriteDockerfileFrom(ctx, content, translator)
				if err != nil {
					pipeWriter.CloseWithError(err)
					return
				}
				hdr.Size = int64(len(newDockerfile))
				content = bytes.NewBuffer(newDockerfile)
			}

			if err := tarWriter.WriteHeader(hdr); err != nil {
				pipeWriter.CloseWithError(err)
				return
			}

			if _, err := io.Copy(tarWriter, content); err != nil {
				pipeWriter.CloseWithError(err)
				return
			}
		}
	}()

	return pipeReader
}
