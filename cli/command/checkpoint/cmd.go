// +build !experimental

package checkpoint

import (
	"github.com/tiborvass/docker/cli/command"
	"github.com/spf13/cobra"
)

// NewCheckpointCommand returns a cobra command for `checkpoint` subcommands
func NewCheckpointCommand(rootCmd *cobra.Command, dockerCli *command.DockerCli) {
}
