package swarm

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newUpdateCommand(dockerCli *command.DockerCli) *cobra.Command {
	opts := swarmOptions{}

	cmd := &cobra.Command{
		Use:   "update [OPTIONS]",
		Short: "Update the swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(dockerCli, cmd.Flags(), opts)
		},
	}

	addSwarmFlags(cmd.Flags(), &opts)
	return cmd
}

func runUpdate(dockerCli *command.DockerCli, flags *pflag.FlagSet, opts swarmOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	var updateFlags swarm.UpdateFlags

	swarm, err := client.SwarmInspect(ctx)
	if err != nil {
		return err
	}

	prevAutoLock := swarm.Spec.EncryptionConfig.AutoLockManagers

	opts.mergeSwarmSpec(&swarm.Spec, flags)

	curAutoLock := swarm.Spec.EncryptionConfig.AutoLockManagers

	err = client.SwarmUpdate(ctx, swarm.Version, swarm.Spec, updateFlags)
	if err != nil {
		return err
	}

	fmt.Fprintln(dockerCli.Out(), "Swarm updated.")

	if curAutoLock && !prevAutoLock {
		unlockKeyResp, err := client.SwarmGetUnlockKey(ctx)
		if err != nil {
			return errors.Wrap(err, "could not fetch unlock key")
		}
		printUnlockCommand(ctx, dockerCli, unlockKeyResp.UnlockKey)
	}

	return nil
}
