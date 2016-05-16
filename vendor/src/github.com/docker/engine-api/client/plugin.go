package client

import (
	"encoding/json"

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

func (cli *Client) PluginInstall(ctx context.Context, name string, registryAuth string) error {
	headers := map[string][]string{"X-Registry-Auth": {registryAuth}}
	resp, err := cli.post(ctx, "/plugins/"+name+"/install", nil, nil, headers)
	ensureReaderClosed(resp)
	return err
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
