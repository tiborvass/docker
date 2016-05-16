// +build !experimental

package plugin

import (
	"github.com/docker/docker/api/client"
	"github.com/spf13/cobra"
)

func experimentalPushCommand(cmd *cobra.Command, dockerCli *client.DockerCli) {}
