package client

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/client/lib"
	Cli "github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/opts"
	"github.com/tiborvass/docker/pkg/jsonmessage"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/pkg/urlutil"
	"github.com/tiborvass/docker/registry"
)

// CmdImport creates an empty filesystem image, imports the contents of the tarball into the image, and optionally tags the image.
//
// The URL argument is the address of a tarball (.tar, .tar.gz, .tgz, .bzip, .tar.xz, .txz) file or a path to local file relative to docker client. If the URL is '-', then the tar file is read from STDIN.
//
// Usage: docker import [OPTIONS] file|URL|- [REPOSITORY[:TAG]]
func (cli *DockerCli) CmdImport(args ...string) error {
	cmd := Cli.Subcmd("import", []string{"file|URL|- [REPOSITORY[:TAG]]"}, Cli.DockerCommands["import"].Description, true)
	flChanges := opts.NewListOpts(nil)
	cmd.Var(&flChanges, []string{"c", "-change"}, "Apply Dockerfile instruction to the created image")
	message := cmd.String([]string{"m", "-message"}, "", "Set commit message for imported image")
	cmd.Require(flag.Min, 1)

	cmd.ParseFlags(args, true)

	var (
		in         io.Reader
		tag        string
		src        = cmd.Arg(0)
		srcName    = src
		repository = cmd.Arg(1)
		changes    = flChanges.GetAll()
	)

	if cmd.NArg() == 3 {
		fmt.Fprintf(cli.err, "[DEPRECATED] The format 'file|URL|- [REPOSITORY [TAG]]' has been deprecated. Please use file|URL|- [REPOSITORY[:TAG]]\n")
		tag = cmd.Arg(2)
	}

	if repository != "" {
		//Check if the given image name can be resolved
		ref, err := reference.ParseNamed(repository)
		if err != nil {
			return err
		}
		if err := registry.ValidateRepositoryName(ref); err != nil {
			return err
		}
	}

	if src == "-" {
		in = cli.in
	} else if !urlutil.IsURL(src) {
		srcName = "-"
		file, err := os.Open(src)
		if err != nil {
			return err
		}
		defer file.Close()
		in = file

	}

	options := lib.ImportImageOptions{
		Source:         in,
		SourceName:     srcName,
		RepositoryName: repository,
		Message:        *message,
		Tag:            tag,
		Changes:        changes,
	}

	responseBody, err := cli.client.ImportImage(options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesStream(responseBody, cli.out, cli.outFd, cli.isTerminalOut)
}
