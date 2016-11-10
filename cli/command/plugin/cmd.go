package plugin

import (
	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cli/command"
	"github.com/spf13/cobra"
)

// NewPluginCommand returns a cobra command for `plugin` subcommands
func NewPluginCommand(dockerCli *command.DockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins",
		Args:  cli.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.SetOutput(dockerCli.Err())
			cmd.HelpFunc()(cmd, args)
		},
		Tags: map[string]string{"experimental": ""},
	}

	cmd.AddCommand(
		newDisableCommand(dockerCli),
		newEnableCommand(dockerCli),
		newInspectCommand(dockerCli),
		newInstallCommand(dockerCli),
		newListCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newSetCommand(dockerCli),
		newPushCommand(dockerCli),
		newCreateCommand(dockerCli),
	)
	return cmd
}
