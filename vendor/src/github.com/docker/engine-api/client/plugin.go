package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/pkg/term"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

// PluginList returns the plugins configured in the docker host.
func (cli *Client) PluginList(ctx context.Context) (types.PluginsListResponse, error) {
	var plugins types.PluginsListResponse
	resp, err := cli.get(ctx, "/plugins", nil, nil)
	if err != nil {
		return plugins, err
	}

	err = json.NewDecoder(resp.body).Decode(&plugins)
	ensureReaderClosed(resp)
	return plugins, err
}

// PluginRemove removes a plugin from the docker host.
func (cli *Client) PluginRemove(ctx context.Context, name string) error {
	resp, err := cli.delete(ctx, "/plugins/"+name, nil, nil)
	ensureReaderClosed(resp)
	return err
}

func (cli *Client) PluginEnable(ctx context.Context, name string) error {
	resp, err := cli.post(ctx, "/plugins/"+name+"/enable", nil, nil, nil)
	ensureReaderClosed(resp)
	return err
}

func (cli *Client) PluginDisable(ctx context.Context, name string) error {
	resp, err := cli.post(ctx, "/plugins/"+name+"/disable", nil, nil, nil)
	ensureReaderClosed(resp)
	return err
}

func (cli *Client) PluginInstall(ctx context.Context, name, registryAuth string, acceptAllPermissions bool, in io.ReadCloser, out io.Writer) error {
	headers := map[string][]string{"X-Registry-Auth": {registryAuth}}
	resp, err := cli.post(ctx, "/plugins/"+name+"/pull", nil, nil, headers)
	if err != nil {
		ensureReaderClosed(resp)
		return err
	}
	var privileges *types.PluginPrivileges
	if err := json.NewDecoder(resp.body).Decode(&privileges); err != nil {
		return err
	}
	ensureReaderClosed(resp)

	if !acceptAllPermissions && privileges != nil {
		_, isTerminalIn := term.GetFdInfo(in)
		_, isTerminalOut := term.GetFdInfo(out)
		if !isTerminalIn || !isTerminalOut {
			// TODO: should we return a need-terminal error?
			return pluginPermissionDenied{name}
		}

		fmt.Fprintf(out, "Plugin %q requested the following privileges:\n", name)
		// host network
		if privileges.Network != nil {
			fmt.Fprintln(out, " - Networking:", *privileges.Network)
		}

		// host fs
		for _, mount := range privileges.Mounts {
			fmt.Fprintln(out, " - Mounting host path:", mount)
		}

		// host devices
		for _, dev := range privileges.Devices {
			fmt.Fprintln(out, " - Mounting host device:", dev)
		}

		// additional capabilities
		if len(privileges.Capabilities) > 0 {
			fmt.Fprintln(out, " - Adding capabilities:", privileges.Capabilities)
		}

		fmt.Fprint(out, "Do you grant the above permissions? [y/N] ")
		reader := bufio.NewReader(in)
		line, _, err := reader.ReadLine()
		if err != nil {
			return err
		}
		if strings.ToLower(string(line)) != "y" {
			return pluginPermissionDenied{name}
		}
	}
	return cli.PluginEnable(ctx, name)
}

func (cli *Client) PluginPush(ctx context.Context, name string, registryAuth string) error {
	headers := map[string][]string{"X-Registry-Auth": {registryAuth}}
	resp, err := cli.post(ctx, "/plugins/"+name+"/push", nil, nil, headers)
	ensureReaderClosed(resp)
	return err
}

func (cli *Client) PluginInspect(ctx context.Context, name string) (*types.Plugin, error) {
	var p types.Plugin
	resp, err := cli.get(ctx, "/plugins/"+name, nil, nil)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(resp.body).Decode(&p)
	ensureReaderClosed(resp)
	return &p, err
}

func (cli *Client) PluginSet(ctx context.Context, name string, args []string) error {
	// TODO: encode args
	resp, err := cli.post(ctx, "/plugins/"+name+"/set", nil, nil, nil)
	ensureReaderClosed(resp)
	return err
}
