package node

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/cli"
	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCli *client.DockerCli) *cobra.Command {
	return &cobra.Command{
		Use:     "rm NODE [NODE...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more nodes from the swarm",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(dockerCli, args)
		},
	}
}

func runRemove(dockerCli *client.DockerCli, args []string) error {
	client := dockerCli.Client()
	ctx := context.Background()
	for _, nodeID := range args {
		err := client.NodeRemove(ctx, nodeID)
		if err != nil {
			return err
		}
		fmt.Fprintf(dockerCli.Out(), "%s\n", nodeID)
	}
	return nil
}
