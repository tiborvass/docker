package swarm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cli/command"
	"golang.org/x/net/context"
)

func newUnlockCommand(dockerCli *command.DockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock swarm",
		Args:  cli.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := dockerCli.Client()
			ctx := context.Background()

			// First see if the node is actually part of a swarm, and if it's is actually locked first.
			// If it's in any other state than locked, don't ask for the key.
			info, err := client.Info(ctx)
			if err != nil {
				return err
			}

			switch info.Swarm.LocalNodeState {
			case swarm.LocalNodeStateInactive:
				return errors.New("Error: This node is not part of a swarm")
			case swarm.LocalNodeStateLocked:
				break
			default:
				return errors.New("Error: swarm is not locked")
			}

			key, err := readKey(dockerCli.In(), "Please enter unlock key: ")
			if err != nil {
				return err
			}
			req := swarm.UnlockRequest{
				UnlockKey: key,
			}

			return client.SwarmUnlock(ctx, req)
		},
	}

	return cmd
}

func readKey(in *command.InStream, prompt string) (string, error) {
	if in.IsTerminal() {
		fmt.Print(prompt)
		dt, err := terminal.ReadPassword(int(in.FD()))
		fmt.Println()
		return string(dt), err
	}
	key, err := bufio.NewReader(in).ReadString('\n')
	if err == io.EOF {
		err = nil
	}
	return strings.TrimSpace(key), err
}
