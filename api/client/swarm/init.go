package swarm

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/cli"
	"github.com/docker/engine-api/types/swarm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type initOptions struct {
	listenAddr      NodeAddrOption
	autoAccept      AutoAcceptOption
	forceNewCluster bool
	secret          string
}

func newInitCommand(dockerCli *client.DockerCli) *cobra.Command {
	opts := initOptions{
		listenAddr: NewListenAddrOption(),
		autoAccept: NewAutoAcceptOption(),
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a Swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(dockerCli, cmd.Flags(), opts)
		},
	}

	flags := cmd.Flags()
	flags.Var(&opts.listenAddr, "listen-addr", "Listen address")
	flags.Var(&opts.autoAccept, flagAutoAccept, "Auto acceptance policy (worker, manager, or none)")
	flags.StringVar(&opts.secret, flagSecret, "", "Set secret value needed to accept nodes into cluster")
	flags.BoolVar(&opts.forceNewCluster, "force-new-cluster", false, "Force create a new cluster from current state.")
	return cmd
}

func runInit(dockerCli *client.DockerCli, flags *pflag.FlagSet, opts initOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	req := swarm.InitRequest{
		ListenAddr:      opts.listenAddr.String(),
		ForceNewCluster: opts.forceNewCluster,
	}

	if flags.Changed(flagSecret) {
		req.Spec.AcceptancePolicy.Policies = opts.autoAccept.Policies(&opts.secret)
	} else {
		req.Spec.AcceptancePolicy.Policies = opts.autoAccept.Policies(nil)
	}
	nodeID, err := client.SwarmInit(ctx, req)
	if err != nil {
		return err
	}
	fmt.Printf("Swarm initialized: current node (%s) is now a manager.\n", nodeID)
	return nil
}
