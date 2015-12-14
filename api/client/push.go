package client

import (
	"errors"
	"fmt"
	"io"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/client/lib"
	"github.com/tiborvass/docker/api/types"
	Cli "github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cliconfig"
	"github.com/tiborvass/docker/pkg/jsonmessage"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/registry"
)

// CmdPush pushes an image or repository to the registry.
//
// Usage: docker push NAME[:TAG]
func (cli *DockerCli) CmdPush(args ...string) error {
	cmd := Cli.Subcmd("push", []string{"NAME[:TAG]"}, Cli.DockerCommands["push"].Description, true)
	addTrustedFlags(cmd, false)
	cmd.Require(flag.Exact, 1)

	cmd.ParseFlags(args, true)

	ref, err := reference.ParseNamed(cmd.Arg(0))
	if err != nil {
		return err
	}

	var tag string
	switch x := ref.(type) {
	case reference.Digested:
		return errors.New("cannot push a digest reference")
	case reference.Tagged:
		tag = x.Tag()
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}
	// Resolve the Auth config relevant for this server
	authConfig := registry.ResolveAuthConfig(cli.configFile.AuthConfigs, repoInfo.Index)
	// If we're not using a custom registry, we know the restrictions
	// applied to repository names and can warn the user in advance.
	// Custom repositories can have different rules, and we must also
	// allow pushing by image ID.
	if repoInfo.Official {
		username := authConfig.Username
		if username == "" {
			username = "<user>"
		}
		return fmt.Errorf("You cannot push a \"root\" repository. Please rename your repository to <user>/<repo> (ex: %s/%s)", username, repoInfo.LocalName)
	}

	requestPrivilege := cli.registryAuthenticationPrivilegedFunc(repoInfo.Index, "push")
	if isTrusted() {
		return cli.trustedPush(repoInfo, tag, authConfig, requestPrivilege)
	}

	return cli.imagePushPrivileged(authConfig, ref.Name(), tag, cli.out, requestPrivilege)
}

func (cli *DockerCli) imagePushPrivileged(authConfig types.AuthConfig, imageID, tag string, outputStream io.Writer, requestPrivilege lib.RequestPrivilegeFunc) error {
	encodedAuth, err := cliconfig.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}
	options := types.ImagePushOptions{
		ImageID:      imageID,
		Tag:          tag,
		RegistryAuth: encodedAuth,
	}

	responseBody, err := cli.client.ImagePush(options, requestPrivilege)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesStream(responseBody, outputStream, cli.outFd, cli.isTerminalOut)
}
