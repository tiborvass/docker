package node

import (
	"fmt"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/tiborvass/docker/cli"
	"github.com/spf13/cobra"
)

func newDemoteCommand(dockerCli *client.DockerCli) *cobra.Command {
	return &cobra.Command{
		Use:   "demote NODE [NODE...]",
		Short: "Demote one or more nodes from manager in the swarm",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemote(dockerCli, args)
		},
	}
}

func runDemote(dockerCli *client.DockerCli, nodes []string) error {
	demote := func(node *swarm.Node) error {
		if node.Spec.Role == swarm.NodeRoleWorker {
			fmt.Fprintf(dockerCli.Out(), "Node %s is already a worker.\n", node.ID)
			return errNoRoleChange
		}
		node.Spec.Role = swarm.NodeRoleWorker
		return nil
	}
	success := func(nodeID string) {
		fmt.Fprintf(dockerCli.Out(), "Manager %s demoted in the swarm.\n", nodeID)
	}
	return updateNodes(dockerCli, nodes, demote, success)
}
