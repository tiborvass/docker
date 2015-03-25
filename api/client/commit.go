package client

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/tiborvass/docker/engine"
	"github.com/tiborvass/docker/opts"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/pkg/parsers"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/runconfig"
	"github.com/tiborvass/docker/utils"
)

// CmdAttach attaches to a running container.
//
// Usage: docker attach [OPTIONS] CONTAINER
func (cli *DockerCli) CmdCommit(args ...string) error {
	cmd := cli.Subcmd("commit", "CONTAINER [REPOSITORY[:TAG]]", "Create a new image from a container's changes", true)
	flPause := cmd.Bool([]string{"p", "-pause"}, true, "Pause container during commit")
	flComment := cmd.String([]string{"m", "-message"}, "", "Commit message")
	flAuthor := cmd.String([]string{"a", "#author", "-author"}, "", "Author (e.g., \"John Hannibal Smith <hannibal@a-team.com>\")")
	flChanges := opts.NewListOpts(nil)
	cmd.Var(&flChanges, []string{"c", "-change"}, "Apply Dockerfile instruction to the created image")
	// FIXME: --run is deprecated, it will be replaced with inline Dockerfile commands.
	flConfig := cmd.String([]string{"#run", "#-run"}, "", "This option is deprecated and will be removed in a future version in favor of inline Dockerfile-compatible commands")
	cmd.Require(flag.Max, 2)
	cmd.Require(flag.Min, 1)
	utils.ParseFlags(cmd, args, true)

	var (
		name            = cmd.Arg(0)
		repository, tag = parsers.ParseRepositoryTag(cmd.Arg(1))
	)

	//Check if the given image name can be resolved
	if repository != "" {
		if err := registry.ValidateRepositoryName(repository); err != nil {
			return err
		}
	}

	v := url.Values{}
	v.Set("container", name)
	v.Set("repo", repository)
	v.Set("tag", tag)
	v.Set("comment", *flComment)
	v.Set("author", *flAuthor)
	for _, change := range flChanges.GetAll() {
		v.Add("changes", change)
	}

	if *flPause != true {
		v.Set("pause", "0")
	}

	var (
		config *runconfig.Config
		env    engine.Env
	)
	if *flConfig != "" {
		config = &runconfig.Config{}
		if err := json.Unmarshal([]byte(*flConfig), config); err != nil {
			return err
		}
	}
	stream, _, err := cli.call("POST", "/commit?"+v.Encode(), config, false)
	if err != nil {
		return err
	}
	if err := env.Decode(stream); err != nil {
		return err
	}

	fmt.Fprintf(cli.out, "%s\n", env.Get("Id"))
	return nil
}
