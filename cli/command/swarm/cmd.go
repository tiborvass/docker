package swarm

import (
	"github.com/spf13/cobra"

	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cli/command"
)

// NewSwarmCommand returns a cobra command for `swarm` subcommands
func NewSwarmCommand(dockerCli *command.DockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swarm",
		Short: "Manage Swarm",
		Args:  cli.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.SetOutput(dockerCli.Err())
			cmd.HelpFunc()(cmd, args)
		},
	}
	cmd.AddCommand(
		newInitCommand(dockerCli),
		newJoinCommand(dockerCli),
		newJoinTokenCommand(dockerCli),
		newUnlockKeyCommand(dockerCli),
		newUpdateCommand(dockerCli),
		newLeaveCommand(dockerCli),
		newUnlockCommand(dockerCli),
	)
	return cmd
}
