// +build experimental

package plugin

import (
	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/cli"
	"github.com/spf13/cobra"
)

func newSetCommand(dockerCli *client.DockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set PLUGIN key1=value1 [key2=value2...]",
		Short: "Change settings for a plugin",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSet(dockerCli, args[0], args[1:])
		},
	}

	return cmd
}

func runSet(dockerCli *client.DockerCli, name string, args []string) error {
	return dockerCli.Client().PluginSet(context.Background(), name, args)
}
