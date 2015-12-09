package client

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/types"
	Cli "github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/opts"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/runconfig"
)

// CmdCommit creates a new image from a container's changes.
//
// Usage: docker commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]
func (cli *DockerCli) CmdCommit(args ...string) error {
	cmd := Cli.Subcmd("commit", []string{"CONTAINER [REPOSITORY[:TAG]]"}, Cli.DockerCommands["commit"].Description, true)
	flPause := cmd.Bool([]string{"p", "-pause"}, true, "Pause container during commit")
	flComment := cmd.String([]string{"m", "-message"}, "", "Commit message")
	flAuthor := cmd.String([]string{"a", "-author"}, "", "Author (e.g., \"John Hannibal Smith <hannibal@a-team.com>\")")
	flChanges := opts.NewListOpts(nil)
	cmd.Var(&flChanges, []string{"c", "-change"}, "Apply Dockerfile instruction to the created image")
	// FIXME: --run is deprecated, it will be replaced with inline Dockerfile commands.
	flConfig := cmd.String([]string{"#-run"}, "", "This option is deprecated and will be removed in a future version in favor of inline Dockerfile-compatible commands")
	cmd.Require(flag.Max, 2)
	cmd.Require(flag.Min, 1)

	cmd.ParseFlags(args, true)

	var (
		name             = cmd.Arg(0)
		repositoryAndTag = cmd.Arg(1)
		repositoryName   string
		tag              string
	)

	//Check if the given image name can be resolved
	if repositoryAndTag != "" {
		ref, err := reference.ParseNamed(repositoryAndTag)
		if err != nil {
			return err
		}
		if err := registry.ValidateRepositoryName(ref); err != nil {
			return err
		}

		repositoryName = ref.Name()

		switch x := ref.(type) {
		case reference.Digested:
			return errors.New("cannot commit to digest reference")
		case reference.Tagged:
			tag = x.Tag()
		}
	}

	var config *runconfig.Config
	if *flConfig != "" {
		config = &runconfig.Config{}
		if err := json.Unmarshal([]byte(*flConfig), config); err != nil {
			return err
		}
	}

	options := types.ContainerCommitOptions{
		ContainerID:    name,
		RepositoryName: repositoryName,
		Tag:            tag,
		Comment:        *flComment,
		Author:         *flAuthor,
		Changes:        flChanges.GetAll(),
		Pause:          *flPause,
		Config:         config,
	}

	response, err := cli.client.ContainerCommit(options)
	if err != nil {
		return err
	}

	fmt.Fprintln(cli.out, response.ID)
	return nil
}
