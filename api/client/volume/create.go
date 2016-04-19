package volume

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/opts"
	runconfigopts "github.com/tiborvass/docker/runconfig/opts"
	"github.com/docker/engine-api/types"
	"github.com/spf13/cobra"
)

type createOptions struct {
	name       string
	driver     string
	driverOpts opts.MapOpts
	labels     []string
}

func newCreateCommand(dockerCli *client.DockerCli) *cobra.Command {
	var opts createOptions

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(dockerCli, opts)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&opts.driver, "driver", "d", "local", "Specify volume driver name")
	flags.StringVar(&opts.name, "name", "", "Specify volume name")
	flags.VarP(&opts.driverOpts, "opt", "o", "Set driver specific options")
	flags.StringSliceVar(&opts.labels, "label", []string{}, "Set metadata for a volume")

	return cmd
}

func runCreate(dockerCli *client.DockerCli, opts createOptions) error {
	client := dockerCli.Client()

	volReq := types.VolumeCreateRequest{
		Driver:     opts.driver,
		DriverOpts: opts.driverOpts.GetAll(),
		Name:       opts.name,
		Labels:     runconfigopts.ConvertKVStringsToMap(opts.labels),
	}

	vol, err := client.VolumeCreate(context.Background(), volReq)
	if err != nil {
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "%s\n", vol.Name)
	return nil
}
