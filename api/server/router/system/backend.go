package system

import (
	"time"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/events"
	"github.com/tiborvass/docker/api/types/filters"
	"golang.org/x/net/context"
)

// Backend is the methods that need to be implemented to provide
// system specific functionality.
type Backend interface {
	SystemInfo() (*types.Info, error)
	SystemVersion() types.Version
	SystemDiskUsage(ctx context.Context) (*types.DiskUsage, error)
	SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{})
	UnsubscribeFromEvents(chan interface{})
	AuthenticateToRegistry(ctx context.Context, authConfig *types.AuthConfig) (string, string, error)
}
