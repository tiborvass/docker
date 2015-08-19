package dockerfile

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/builder"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/ulimit"
	"github.com/docker/docker/runconfig"
)

type Config struct {
	//dockerfile.BuilderConfig

	// only used if Dockerfile has to be extracted from Context
	DockerfileName string

	// TODO: possibly not needed if appropriate context is passed (or deduced)
	//RemoteURL string

	// TODO: not needed if it's only to tag the image: responsibility of the caller
	//RepoName string

	Verbose     bool
	UseCache    bool
	Remove      bool
	ForceRemove bool
	Pull        bool

	// resource constraints
	// TODO: factor out to be reused with Run ?

	Memory       int64
	MemorySwap   int64
	CPUShares    int64
	CPUPeriod    int64
	CPUQuota     int64
	CPUSetCpus   string
	CPUSetMems   string
	CgroupParent string
	Ulimits      []*ulimit.Ulimit
}

type Builder struct {
	*Config

	Stdout io.Writer
	Stderr io.Writer

	// for transactions
	id string

	docker  builder.Docker
	context builder.Context

	dockerfile    *parser.Node
	runConfig     *runconfig.Config // runconfig for cmd, run, entrypoint etc.
	flags         *BFlags
	tmpContainers map[string]struct{}
	image         string // imageID
	noBaseImage   bool
	maintainer    string
	cmdSet        bool
	disableCommit bool
	cacheBusted   bool
}

// if dockerfile is nil, remember to read dockerfile in Build() from Context
func NewBuilder(dockerfile io.ReadCloser, config *Config) (b *Builder, err error) {
	if config == nil {
		config = new(Config)
	}
	b = &Builder{
		Config:        config,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		runConfig:     new(runconfig.Config),
		tmpContainers: map[string]struct{}{},
	}
	if dockerfile != nil {
		b.dockerfile, err = parser.Parse(dockerfile)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

// Run the builder with the context. This is the lynchpin of this package. This
// will (barring errors):
//
// * call readContext() which will set up the temporary directory and unpack
//   the context into it.
// * read the dockerfile
// * parse the dockerfile
// * walk the parse tree and execute it by dispatching to handlers. If Remove
//   or ForceRemove is set, additional cleanup around containers happens after
//   processing.
// * Print a happy message and return the image ID.
//
func (b *Builder) Build(docker builder.Docker, context builder.Context) (builder.ImageID, error) {
	b.docker = docker
	b.context = context

	// only for HTTP
	// sf := streamformatter.NewJSONStreamFormatter()

	// TODO: use reference, and this is only needed to tag after building image,
	// thus this should be responsibility of the caller
	/*
		repoName, tag := parsers.ParseRepositoryTag(b.RepoName)
		if repoName != "" {
			if err := registry.ValidateRepositoryName(repoName); err != nil {
				return img, err
			}
			if len(tag) > 0 {
				if err := tags.ValidateTagName(tag); err != nil {
					return img, err
				}
			}
		}

		// b.RemoteURL is unnecessary since that should deduce what kind of context to pass
		// to Build(). TODO: add DetectContextFromRemoteURL(remoteURL)
		if b.RemoteURL == "" {
			//context = ioutil.NopCloser(buildConfig.Context)
		} else if urlutil.IsGitURL(b.RemoteURL) {
			root, err := utils.GitClone(b.RemoteURL)
			if err != nil {
				return img, err
			}
			defer os.RemoveAll(root)

			c, err := archive.Tar(root, archive.Uncompressed)
			if err != nil {
				return img, err
			}
		} else if urlutil.IsURL(b.RemoteURL) {
			f, err := httputils.Download(b.RemoteURL)
			if err != nil {
				return fmt.Errorf("Error downloading remote context %s: %v", b.RemoteURL, err)
			}
			defer f.Body.Close()
			ct := f.Header.Get("Content-Type")
			clen := f.ContentLength
			contentType, bodyReader, err := inspectResponse(ct, f.Body, clen)

			defer bodyReader.Close()

			if err != nil {
				return fmt.Errorf("Error detecting content type for remote %s: %v", b.RemoteURL, err)
			}
			if contentType == httputils.MimeTypes.TextPlain {
				dockerFile, err := ioutil.ReadAll(bodyReader)
				if err != nil {
					return err
				}

				// When we're downloading just a Dockerfile put it in
				// the default name - don't allow the client to move/specify it
				b.DockerfileName = api.DefaultDockerfileName

				c, err := archive.Generate(b.DockerfileName, string(dockerFile))
				if err != nil {
					return err
				}
				//context = c
			} else {
				// Pass through - this is a pre-packaged context, presumably
				// with a Dockerfile with the right name inside it.
				prCfg := progressreader.Config{
					In:        bodyReader,
					Out:       b.Stdout,
					Formatter: sf,
					Size:      clen,
					NewLines:  true,
					ID:        "Downloading context",
					Action:    b.RemoteURL,
				}
				//context = progressreader.New(prCfg)
			}
		}

		// TODO: should i close context?
		//defer context.Close()

		builder := &builder{
			// HTTP
			OutStream: &streamformatter.StdoutFormatter{
				Writer:          b.Stdout,
				StreamFormatter: sf,
			},
			// HTTP
			ErrStream: &streamformatter.StderrFormatter{
				Writer:          buildConfig.Stdout,
				StreamFormatter: sf,
			},

			Verbose:      !buildConfig.SuppressOutput,
			UtilizeCache: !buildConfig.NoCache,
			Remove:       buildConfig.Remove,
			ForceRemove:  buildConfig.ForceRemove,
			Pull:         buildConfig.Pull,

			// WTF
			OutOld: buildConfig.Stdout,

			// HTTP
			StreamFormatter: sf,

			AuthConfigs:    buildConfig.AuthConfigs,
			dockerfileName: buildConfig.DockerfileName,
			cpuShares:      buildConfig.CPUShares,
			cpuPeriod:      buildConfig.CPUPeriod,
			cpuQuota:       buildConfig.CPUQuota,
			cpuSetCpus:     buildConfig.CPUSetCpus,
			cpuSetMems:     buildConfig.CPUSetMems,
			cgroupParent:   buildConfig.CgroupParent,
			memory:         buildConfig.Memory,
			memorySwap:     buildConfig.MemorySwap,
			ulimits:        buildConfig.Ulimits,

			cancelled: buildConfig.WaitCancelled(),
			id:        stringid.GenerateRandomID(),
		}
	*/

	// TODO: transactions?
	//defer func() {
	//	builder.Daemon.Graph().Release(builder.id, builder.activeImages...)
	//}()

	// TODO: readContext can be implemented with an implementation of Context
	//if err := b.readContext(context); err != nil {
	//	return img, err
	//}

	//defer func() {
	//	if err := os.RemoveAll(b.contextPath); err != nil {
	//		logrus.Debugf("[BUILDER] failed to remove temporary context: %s", err)
	//	}
	//}()

	// If Dockerfile was not parsed yet, extract it from the Context
	if b.dockerfile == nil {
		if err := b.readDockerfile(); err != nil {
			return "", err
		}
	}

	var shortImgID string
	for i, n := range b.dockerfile.Children {
		// TODO: find better way to cancel
		/*
			select {
			case <-b.cancelled:
				logrus.Debug("Builder: build cancelled!")
				fmt.Fprintf(b.OutStream, "Build cancelled")
				return img, fmt.Errorf("Build cancelled")
			default:
				// Not cancelled yet, keep going...
			}
		*/
		if err := b.dispatch(i, n); err != nil {
			if b.ForceRemove {
				b.clearTmp()
			}
			return "", err
		}
		shortImgID = stringid.TruncateID(b.image)
		fmt.Fprintf(b.Stdout, " ---> %s\n", shortImgID)
		if b.Remove {
			b.clearTmp()
		}
	}

	if b.image == "" {
		return "", fmt.Errorf("No image was generated. Is your Dockerfile empty?")
	}

	fmt.Fprintf(b.Stdout, "Successfully built %s\n", shortImgID)
	return builder.ImageID(b.image), nil

	// TODO: tagging should be responsibility of the caller, not the builder
	//if repoName != "" {
	//	return d.Repositories().Tag(repoName, tag, id, true)
	//}
}
