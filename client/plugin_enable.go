package client // import "github.com/tiborvass/docker/client"

import (
	"net/url"
	"strconv"

	"github.com/tiborvass/docker/api/types"
	"golang.org/x/net/context"
)

// PluginEnable enables a plugin
func (cli *Client) PluginEnable(ctx context.Context, name string, options types.PluginEnableOptions) error {
	query := url.Values{}
	query.Set("timeout", strconv.Itoa(options.Timeout))

	resp, err := cli.post(ctx, "/plugins/"+name+"/enable", query, nil, nil)
	ensureReaderClosed(resp)
	return err
}
