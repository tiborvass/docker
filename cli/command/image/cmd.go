package image

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cli/command"
)

// NewImageCommand returns a cobra command for `image` subcommands
func NewImageCommand(dockerCli *command.DockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage Docker images",
		Args:  cli.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(dockerCli.Err(), "\n"+cmd.UsageString())
		},
	}
	cmd.AddCommand(
		NewBuildCommand(dockerCli),
		NewHistoryCommand(dockerCli),
		NewImportCommand(dockerCli),
		NewLoadCommand(dockerCli),
		NewPullCommand(dockerCli),
		NewPushCommand(dockerCli),
		NewSaveCommand(dockerCli),
		NewSearchCommand(dockerCli),
		NewTagCommand(dockerCli),
		newListCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newInspectCommand(dockerCli),
	)
	return cmd
}
