package client

import (
	"fmt"

	Cli "github.com/tiborvass/docker/cli"
	flag "github.com/tiborvass/docker/pkg/mflag"
)

// CmdKill kills one or more running container using SIGKILL or a specified signal.
//
// Usage: docker kill [OPTIONS] CONTAINER [CONTAINER...]
func (cli *DockerCli) CmdKill(args ...string) error {
	cmd := Cli.Subcmd("kill", []string{"CONTAINER [CONTAINER...]"}, Cli.DockerCommands["kill"].Description, true)
	signal := cmd.String([]string{"s", "-signal"}, "KILL", "Signal to send to the container")
	cmd.Require(flag.Min, 1)

	cmd.ParseFlags(args, true)

	var errNames []string
	for _, name := range cmd.Args() {
		if err := cli.client.ContainerKill(name, *signal); err != nil {
			fmt.Fprintf(cli.err, "%s\n", err)
			errNames = append(errNames, name)
		} else {
			fmt.Fprintf(cli.out, "%s\n", name)
		}
	}
	if len(errNames) > 0 {
		return fmt.Errorf("Error: failed to kill containers: %v", errNames)
	}
	return nil
}
