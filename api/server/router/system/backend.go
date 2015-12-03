package system

import (
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/cliconfig"
	"github.com/tiborvass/docker/pkg/jsonmessage"
	"github.com/tiborvass/docker/pkg/parsers/filters"
)

// Backend is the methods that need to be implemented to provide
// system specific functionality.
type Backend interface {
	SystemInfo() (*types.Info, error)
	SystemVersion() types.Version
	SubscribeToEvents(since, sinceNano int64, ef filters.Args) ([]*jsonmessage.JSONMessage, chan interface{})
	UnsubscribeFromEvents(chan interface{})
	AuthenticateToRegistry(authConfig *cliconfig.AuthConfig) (string, error)
}
