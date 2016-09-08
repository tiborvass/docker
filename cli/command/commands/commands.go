package commands

import (
	"github.com/tiborvass/docker/cli/command"
	"github.com/tiborvass/docker/cli/command/container"
	"github.com/tiborvass/docker/cli/command/image"
	"github.com/tiborvass/docker/cli/command/network"
	"github.com/tiborvass/docker/cli/command/node"
	"github.com/tiborvass/docker/cli/command/plugin"
	"github.com/tiborvass/docker/cli/command/registry"
	"github.com/tiborvass/docker/cli/command/service"
	"github.com/tiborvass/docker/cli/command/stack"
	"github.com/tiborvass/docker/cli/command/swarm"
	"github.com/tiborvass/docker/cli/command/system"
	"github.com/tiborvass/docker/cli/command/volume"
	"github.com/spf13/cobra"
)

// AddCommands adds all the commands from api/client to the root command
func AddCommands(cmd *cobra.Command, dockerCli *command.DockerCli) {
	cmd.AddCommand(
		node.NewNodeCommand(dockerCli),
		service.NewServiceCommand(dockerCli),
		stack.NewStackCommand(dockerCli),
		stack.NewTopLevelDeployCommand(dockerCli),
		swarm.NewSwarmCommand(dockerCli),
		container.NewAttachCommand(dockerCli),
		container.NewCommitCommand(dockerCli),
		container.NewCopyCommand(dockerCli),
		container.NewCreateCommand(dockerCli),
		container.NewDiffCommand(dockerCli),
		container.NewExecCommand(dockerCli),
		container.NewExportCommand(dockerCli),
		container.NewKillCommand(dockerCli),
		container.NewLogsCommand(dockerCli),
		container.NewPauseCommand(dockerCli),
		container.NewPortCommand(dockerCli),
		container.NewPsCommand(dockerCli),
		container.NewRenameCommand(dockerCli),
		container.NewRestartCommand(dockerCli),
		container.NewRmCommand(dockerCli),
		container.NewRunCommand(dockerCli),
		container.NewStartCommand(dockerCli),
		container.NewStatsCommand(dockerCli),
		container.NewStopCommand(dockerCli),
		container.NewTopCommand(dockerCli),
		container.NewUnpauseCommand(dockerCli),
		container.NewUpdateCommand(dockerCli),
		container.NewWaitCommand(dockerCli),
		image.NewBuildCommand(dockerCli),
		image.NewHistoryCommand(dockerCli),
		image.NewImagesCommand(dockerCli),
		image.NewLoadCommand(dockerCli),
		image.NewRemoveCommand(dockerCli),
		image.NewSaveCommand(dockerCli),
		image.NewPullCommand(dockerCli),
		image.NewPushCommand(dockerCli),
		image.NewSearchCommand(dockerCli),
		image.NewImportCommand(dockerCli),
		image.NewTagCommand(dockerCli),
		network.NewNetworkCommand(dockerCli),
		system.NewEventsCommand(dockerCli),
		system.NewInspectCommand(dockerCli),
		registry.NewLoginCommand(dockerCli),
		registry.NewLogoutCommand(dockerCli),
		system.NewVersionCommand(dockerCli),
		volume.NewVolumeCommand(dockerCli),
		system.NewInfoCommand(dockerCli),
	)
	plugin.NewPluginCommand(cmd, dockerCli)
}
