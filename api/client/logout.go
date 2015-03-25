package client

import (
	"fmt"

	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/utils"
)

// 'docker logout': log out a user from a registry service.
func (cli *DockerCli) CmdLogout(args ...string) error {
	cmd := cli.Subcmd("logout", "[SERVER]", "Log out from a Docker registry, if no server is\nspecified \""+registry.IndexServerAddress()+"\" is the default.", true)
	cmd.Require(flag.Max, 1)

	utils.ParseFlags(cmd, args, false)
	serverAddress := registry.IndexServerAddress()
	if len(cmd.Args()) > 0 {
		serverAddress = cmd.Arg(0)
	}

	cli.LoadConfigFile()
	if _, ok := cli.configFile.Configs[serverAddress]; !ok {
		fmt.Fprintf(cli.out, "Not logged in to %s\n", serverAddress)
	} else {
		fmt.Fprintf(cli.out, "Remove login credentials for %s\n", serverAddress)
		delete(cli.configFile.Configs, serverAddress)

		if err := registry.SaveConfig(cli.configFile); err != nil {
			return fmt.Errorf("Failed to save docker config: %v", err)
		}
	}
	return nil
}
