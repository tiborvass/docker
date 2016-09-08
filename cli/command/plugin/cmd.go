// +build !experimental

package plugin

import (
	"github.com/tiborvass/docker/cli/command"
	"github.com/spf13/cobra"
)

// NewPluginCommand returns a cobra command for `plugin` subcommands
func NewPluginCommand(cmd *cobra.Command, dockerCli *command.DockerCli) {
}
