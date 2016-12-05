package plugin

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cli/command"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/tiborvass/docker/pkg/stringutils"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type listOptions struct {
	noTrunc bool
}

func newListCommand(dockerCli *command.DockerCli) *cobra.Command {
	var opts listOptions

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Short:   "List plugins",
		Aliases: []string{"list"},
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli, opts)
		},
	}

	flags := cmd.Flags()

	flags.BoolVar(&opts.noTrunc, "no-trunc", false, "Don't truncate output")

	return cmd
}

func runList(dockerCli *command.DockerCli, opts listOptions) error {
	plugins, err := dockerCli.Client().PluginList(context.Background())
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(dockerCli.Out(), 20, 1, 3, ' ', 0)
	fmt.Fprintf(w, "ID \tNAME \tTAG \tDESCRIPTION\tENABLED")
	fmt.Fprintf(w, "\n")

	for _, p := range plugins {
		id := p.ID
		desc := strings.Replace(p.Config.Description, "\n", " ", -1)
		desc = strings.Replace(desc, "\r", " ", -1)
		if !opts.noTrunc {
			id = stringid.TruncateID(p.ID)
			desc = stringutils.Ellipsis(desc, 45)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\n", id, p.Name, p.Tag, desc, p.Enabled)
	}
	w.Flush()
	return nil
}
