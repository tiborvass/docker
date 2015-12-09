package client

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/client/lib"
	"github.com/tiborvass/docker/api/types"
	Cli "github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/pkg/jsonmessage"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/runconfig"
	tagpkg "github.com/tiborvass/docker/tag"
)

func (cli *DockerCli) pullImage(image string) error {
	return cli.pullImageCustomOut(image, cli.out)
}

func (cli *DockerCli) pullImageCustomOut(image string, out io.Writer) error {
	ref, err := reference.ParseNamed(image)
	if err != nil {
		return err
	}

	var tag string
	switch x := ref.(type) {
	case reference.Digested:
		tag = x.Digest().String()
	case reference.Tagged:
		tag = x.Tag()
	default:
		// pull only the image tagged 'latest' if no tag was specified
		tag = tagpkg.DefaultTag
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}

	// Resolve the Auth config relevant for this server
	encodedAuth, err := cli.encodeRegistryAuth(repoInfo.Index)
	if err != nil {
		return err
	}

	options := types.ImageCreateOptions{
		Parent:       ref.Name(),
		Tag:          tag,
		RegistryAuth: encodedAuth,
	}

	responseBody, err := cli.client.ImageCreate(options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesStream(responseBody, out, cli.outFd, cli.isTerminalOut)
}

type cidFile struct {
	path    string
	file    *os.File
	written bool
}

func newCIDFile(path string) (*cidFile, error) {
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("Container ID file found, make sure the other container isn't running or delete %s", path)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the container ID file: %s", err)
	}

	return &cidFile{path: path, file: f}, nil
}

func (cli *DockerCli) createContainer(config *runconfig.Config, hostConfig *runconfig.HostConfig, cidfile, name string) (*types.ContainerCreateResponse, error) {
	mergedConfig := runconfig.MergeConfigs(config, hostConfig)

	var containerIDFile *cidFile
	if cidfile != "" {
		var err error
		if containerIDFile, err = newCIDFile(cidfile); err != nil {
			return nil, err
		}
		defer containerIDFile.Close()
	}

	ref, err := reference.ParseNamed(config.Image)
	if err != nil {
		return nil, err
	}

	isDigested := false
	switch ref.(type) {
	case reference.Tagged:
	case reference.Digested:
		isDigested = true
	default:
		ref, err = reference.WithTag(ref, tagpkg.DefaultTag)
		if err != nil {
			return nil, err
		}
	}

	var trustedRef reference.Canonical

	if isTrusted() && !isDigested {
		var err error
		trustedRef, err = cli.trustedReference(ref.(reference.NamedTagged))
		if err != nil {
			return nil, err
		}
		config.Image = trustedRef.String()
	}

	//create the container
	response, err := cli.client.ContainerCreate(mergedConfig, name)
	//if image not found try to pull it
	if err != nil {
		if lib.IsErrImageNotFound(err) {
			fmt.Fprintf(cli.err, "Unable to find image '%s' locally\n", ref.String())

			// we don't want to write to stdout anything apart from container.ID
			if err = cli.pullImageCustomOut(config.Image, cli.err); err != nil {
				return nil, err
			}
			if trustedRef != nil && !isDigested {
				if err := cli.tagTrusted(trustedRef, ref.(reference.NamedTagged)); err != nil {
					return nil, err
				}
			}
			// Retry
			var retryErr error
			response, retryErr = cli.client.ContainerCreate(mergedConfig, name)
			if retryErr != nil {
				return nil, retryErr
			}
		} else {
			return nil, err
		}
	}

	for _, warning := range response.Warnings {
		fmt.Fprintf(cli.err, "WARNING: %s\n", warning)
	}
	if containerIDFile != nil {
		if err = containerIDFile.Write(response.ID); err != nil {
			return nil, err
		}
	}
	return &response, nil
}

// CmdCreate creates a new container from a given image.
//
// Usage: docker create [OPTIONS] IMAGE [COMMAND] [ARG...]
func (cli *DockerCli) CmdCreate(args ...string) error {
	cmd := Cli.Subcmd("create", []string{"IMAGE [COMMAND] [ARG...]"}, Cli.DockerCommands["create"].Description, true)
	addTrustedFlags(cmd, true)

	// These are flags not stored in Config/HostConfig
	var (
		flName = cmd.String([]string{"-name"}, "", "Assign a name to the container")
	)

	config, hostConfig, cmd, err := runconfig.Parse(cmd, args)
	if err != nil {
		cmd.ReportError(err.Error(), true)
		os.Exit(1)
	}
	if config.Image == "" {
		cmd.Usage()
		return nil
	}
	response, err := cli.createContainer(config, hostConfig, hostConfig.ContainerIDFile, *flName)
	if err != nil {
		return err
	}
	fmt.Fprintf(cli.out, "%s\n", response.ID)
	return nil
}
