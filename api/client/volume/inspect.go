package volume

import (
	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/api/client/inspect"
	"github.com/tiborvass/docker/cli"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	format string
	names  []string
}

func newInspectCommand(dockerCli *client.DockerCli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] VOLUME [VOLUME...]",
		Short: "Display detailed information on one or more volumes",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runInspect(dockerCli, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", "Format the output using the given go template")

	return cmd
}

func runInspect(dockerCli *client.DockerCli, opts inspectOptions) error {
	client := dockerCli.Client()

	ctx := context.Background()

	getVolFunc := func(name string) (interface{}, []byte, error) {
		i, err := client.VolumeInspect(ctx, name)
		return i, nil, err
	}

	return inspect.Inspect(dockerCli.Out(), opts.names, opts.format, getVolFunc)
}
