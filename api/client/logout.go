package client

import (
	"fmt"

	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/registry"
)

// CmdLogout logs a user out from a Docker registry.
//
// If no server is specified, the user will be logged out from the registry's index server.
//
// Usage: docker logout [SERVER]
func (cli *DockerCli) CmdLogout(args ...string) error {
	cmd := cli.Subcmd("logout", []string{"[SERVER]"}, "Log out from a Docker registry, if no server is\nspecified \""+registry.IndexServerAddress()+"\" is the default.", true)
	cmd.Require(flag.Max, 1)

	cmd.ParseFlags(args, true)
	serverAddress := registry.IndexServerAddress()
	if len(cmd.Args()) > 0 {
		serverAddress = cmd.Arg(0)
	}

	if _, ok := cli.configFile.AuthConfigs[serverAddress]; !ok {
		fmt.Fprintf(cli.out, "Not logged in to %s\n", serverAddress)
	} else {
		fmt.Fprintf(cli.out, "Remove login credentials for %s\n", serverAddress)
		delete(cli.configFile.AuthConfigs, serverAddress)

		if err := cli.configFile.Save(); err != nil {
			return fmt.Errorf("Failed to save docker config: %v", err)
		}
	}
	return nil
}
