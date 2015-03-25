package client

import (
	"fmt"

	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/utils"
)

// 'docker kill NAME' kills a running container
func (cli *DockerCli) CmdKill(args ...string) error {
	cmd := cli.Subcmd("kill", "CONTAINER [CONTAINER...]", "Kill a running container using SIGKILL or a specified signal", true)
	signal := cmd.String([]string{"s", "-signal"}, "KILL", "Signal to send to the container")
	cmd.Require(flag.Min, 1)

	utils.ParseFlags(cmd, args, true)

	var encounteredError error
	for _, name := range cmd.Args() {
		if _, _, err := readBody(cli.call("POST", fmt.Sprintf("/containers/%s/kill?signal=%s", name, *signal), nil, false)); err != nil {
			fmt.Fprintf(cli.err, "%s\n", err)
			encounteredError = fmt.Errorf("Error: failed to kill one or more containers")
		} else {
			fmt.Fprintf(cli.out, "%s\n", name)
		}
	}
	return encounteredError
}
