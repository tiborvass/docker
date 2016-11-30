package stack

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cli/command"
	"github.com/tiborvass/docker/cli/command/idresolver"
	"github.com/tiborvass/docker/cli/command/task"
	"github.com/tiborvass/docker/opts"
	"github.com/spf13/cobra"
)

type psOptions struct {
	all       bool
	filter    opts.FilterOpt
	noTrunc   bool
	namespace string
	noResolve bool
}

func newPsCommand(dockerCli *command.DockerCli) *cobra.Command {
	opts := psOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS] STACK",
		Short: "List the tasks in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.namespace = args[0]
			return runPS(dockerCli, opts)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.all, "all", "a", false, "Display all tasks")
	flags.BoolVar(&opts.noTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&opts.noResolve, "no-resolve", false, "Do not map IDs to Names")
	flags.VarP(&opts.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

func runPS(dockerCli *command.DockerCli, opts psOptions) error {
	namespace := opts.namespace
	client := dockerCli.Client()
	ctx := context.Background()

	filter := getStackFilterFromOpt(opts.namespace, opts.filter)
	if !opts.all && !filter.Include("desired-state") {
		filter.Add("desired-state", string(swarm.TaskStateRunning))
		filter.Add("desired-state", string(swarm.TaskStateAccepted))
	}

	tasks, err := client.TaskList(ctx, types.TaskListOptions{Filters: filter})
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		fmt.Fprintf(dockerCli.Out(), "Nothing found in stack: %s\n", namespace)
		return nil
	}

	return task.Print(dockerCli, ctx, tasks, idresolver.New(client, opts.noResolve), opts.noTrunc)
}
