package builder

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/engine"
	"github.com/tiborvass/docker/graph"
	"github.com/tiborvass/docker/pkg/archive"
	"github.com/tiborvass/docker/pkg/parsers"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/utils"
)

type BuilderJob struct {
	Engine *engine.Engine
	Daemon *daemon.Daemon
}

func (b *BuilderJob) Install() {
	b.Engine.Register("build", b.CmdBuild)
}

func (b *BuilderJob) CmdBuild(job *engine.Job) engine.Status {
	if len(job.Args) != 0 {
		return job.Errorf("Usage: %s\n", job.Name)
	}
	var (
		remoteURL      = job.Getenv("remote")
		repoName       = job.Getenv("t")
		suppressOutput = job.GetenvBool("q")
		noCache        = job.GetenvBool("nocache")
		rm             = job.GetenvBool("rm")
		forceRm        = job.GetenvBool("forcerm")
		authConfig     = &registry.AuthConfig{}
		configFile     = &registry.ConfigFile{}
		tag            string
		context        io.ReadCloser
	)
	job.GetenvJson("authConfig", authConfig)
	job.GetenvJson("configFile", configFile)

	repoName, tag = parsers.ParseRepositoryTag(repoName)
	if repoName != "" {
		if _, _, err := registry.ResolveRepositoryName(repoName); err != nil {
			return job.Error(err)
		}
		if len(tag) > 0 {
			if err := graph.ValidateTagName(tag); err != nil {
				return job.Error(err)
			}
		}
	}

	if remoteURL == "" {
		context = ioutil.NopCloser(job.Stdin)
	} else if utils.IsGIT(remoteURL) {
		if !strings.HasPrefix(remoteURL, "git://") {
			remoteURL = "https://" + remoteURL
		}
		root, err := ioutil.TempDir("", "docker-build-git")
		if err != nil {
			return job.Error(err)
		}
		defer os.RemoveAll(root)

		if output, err := exec.Command("git", "clone", "--recursive", remoteURL, root).CombinedOutput(); err != nil {
			return job.Errorf("Error trying to use git: %s (%s)", err, output)
		}

		c, err := archive.Tar(root, archive.Uncompressed)
		if err != nil {
			return job.Error(err)
		}
		context = c
	} else if utils.IsURL(remoteURL) {
		f, err := utils.Download(remoteURL)
		if err != nil {
			return job.Error(err)
		}
		defer f.Body.Close()
		dockerFile, err := ioutil.ReadAll(f.Body)
		if err != nil {
			return job.Error(err)
		}
		c, err := archive.Generate("Dockerfile", string(dockerFile))
		if err != nil {
			return job.Error(err)
		}
		context = c
	}
	defer context.Close()

	sf := utils.NewStreamFormatter(job.GetenvBool("json"))

	builder := &Builder{
		Daemon: b.Daemon,
		Engine: b.Engine,
		OutStream: &utils.StdoutFormater{
			Writer:          job.Stdout,
			StreamFormatter: sf,
		},
		ErrStream: &utils.StderrFormater{
			Writer:          job.Stdout,
			StreamFormatter: sf,
		},
		Verbose:         !suppressOutput,
		UtilizeCache:    !noCache,
		Remove:          rm,
		ForceRemove:     forceRm,
		OutOld:          job.Stdout,
		StreamFormatter: sf,
		AuthConfig:      authConfig,
		AuthConfigFile:  configFile,
	}

	id, err := builder.Run(context)
	if err != nil {
		return job.Error(err)
	}

	if repoName != "" {
		b.Daemon.Repositories().Set(repoName, tag, id, true)
	}
	return engine.StatusOK
}
