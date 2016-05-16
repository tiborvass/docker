package plugin

import (
	"net/http"

	enginetypes "github.com/docker/engine-api/types"
)

// Backend for Plugin
type Backend interface {
	Disable(name string) error
	Enable(name string) error
	List() ([]enginetypes.Plugin, error)
	Inspect(name string) (enginetypes.Plugin, error)
	Install(name string, metaHeaders http.Header, authConfig *enginetypes.AuthConfig) error
	Remove(name string) error
	Set(name string, args []string) error
	Push(name string, metaHeaders http.Header, authConfig *enginetypes.AuthConfig) error
}
