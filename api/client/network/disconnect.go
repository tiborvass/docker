package network

import (
	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/cli"
	"github.com/spf13/cobra"
)

type disconnectOptions struct {
	network   string
	container string
	force     bool
}

func newDisconnectCommand(dockerCli *client.DockerCli) *cobra.Command {
	opts := disconnectOptions{}

	cmd := &cobra.Command{
		Use:   "disconnect [OPTIONS] NETWORK CONTAINER",
		Short: "Disconnect a container from a network",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.network = args[0]
			opts.container = args[1]
			return runDisconnect(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the container to disconnect from a network")

	return cmd
}

func runDisconnect(dockerCli *client.DockerCli, opts disconnectOptions) error {
	client := dockerCli.Client()

	return client.NetworkDisconnect(context.Background(), opts.network, opts.container, opts.force)
}
