package image

import (
	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/pkg/jsonmessage"
	"github.com/tiborvass/docker/reference"
	"github.com/tiborvass/docker/registry"
	"github.com/spf13/cobra"
)

// NewPushCommand creates a new `docker push` command
func NewPushCommand(dockerCli *client.DockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push NAME[:TAG]",
		Short: "Push an image or a repository to a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(dockerCli, args[0])
		},
	}

	flags := cmd.Flags()

	client.AddTrustedFlags(flags, true)

	return cmd
}

func runPush(dockerCli *client.DockerCli, remote string) error {
	ref, err := reference.ParseNamed(remote)
	if err != nil {
		return err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve the Auth config relevant for this server
	authConfig := dockerCli.ResolveAuthConfig(ctx, repoInfo.Index)
	requestPrivilege := dockerCli.RegistryAuthenticationPrivilegedFunc(repoInfo.Index, "push")

	if client.IsTrusted() {
		return dockerCli.TrustedPush(ctx, repoInfo, ref, authConfig, requestPrivilege)
	}

	responseBody, err := dockerCli.ImagePushPrivileged(ctx, authConfig, ref.String(), requestPrivilege)
	if err != nil {
		return err
	}

	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesStream(responseBody, dockerCli.Out(), dockerCli.OutFd(), dockerCli.IsTerminalOut(), nil)
}
