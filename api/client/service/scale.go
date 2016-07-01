package service

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"github.com/tiborvass/docker/api/client"
	"github.com/tiborvass/docker/cli"
	"github.com/spf13/cobra"
)

func newScaleCommand(dockerCli *client.DockerCli) *cobra.Command {
	return &cobra.Command{
		Use:   "scale SERVICE=REPLICAS [SERVICE=REPLICAS...]",
		Short: "Scale one or multiple services",
		Args:  scaleArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScale(dockerCli, args)
		},
	}
}

func scaleArgs(cmd *cobra.Command, args []string) error {
	if err := cli.RequiresMinArgs(1)(cmd, args); err != nil {
		return err
	}
	for _, arg := range args {
		if parts := strings.SplitN(arg, "=", 2); len(parts) != 2 {
			return fmt.Errorf(
				"Invalid scale specifier '%s'.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
				arg,
				cmd.CommandPath(),
				cmd.UseLine(),
				cmd.Short,
			)
		}
	}
	return nil
}

func runScale(dockerCli *client.DockerCli, args []string) error {
	var errors []string
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		serviceID, scale := parts[0], parts[1]
		if err := runServiceScale(dockerCli, serviceID, scale); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", serviceID, err.Error()))
		}
	}

	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errors, "\n"))
}

func runServiceScale(dockerCli *client.DockerCli, serviceID string, scale string) error {
	client := dockerCli.Client()
	ctx := context.Background()
	headers := map[string][]string{}

	service, _, err := client.ServiceInspectWithRaw(ctx, serviceID)

	if err != nil {
		return err
	}

	// TODO(nishanttotla): Is this the best way to get the image?
	image := service.Spec.TaskTemplate.ContainerSpec.Image
	if image != "" {
		// Retrieve encoded auth token from the image reference
		encodedAuth, err := dockerCli.RetrieveAuthTokenFromImage(ctx, image)
		if err != nil {
			return err
		}
		headers = map[string][]string{
			"x-registry-auth": {encodedAuth},
		}
	}

	serviceMode := &service.Spec.Mode
	if serviceMode.Replicated == nil {
		return fmt.Errorf("scale can only be used with replicated mode")
	}
	uintScale, err := strconv.ParseUint(scale, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid replicas value %s: %s", scale, err.Error())
	}
	serviceMode.Replicated.Replicas = &uintScale

	err = client.ServiceUpdate(ctx, service.ID, service.Version, service.Spec, headers)
	if err != nil {
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "%s scaled to %s\n", serviceID, scale)
	return nil
}
