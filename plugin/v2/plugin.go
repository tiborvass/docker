package v2

import (
	"sync"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/pkg/plugins"
	"github.com/tiborvass/docker/restartmanager"
)

// Plugin represents an individual plugin.
type Plugin struct {
	sync.RWMutex
	PluginObj         types.Plugin                  `json:"plugin"`
	PClient           *plugins.Client               `json:"-"`
	RestartManager    restartmanager.RestartManager `json:"-"`
	RuntimeSourcePath string                        `json:"-"`
	ExitChan          chan bool                     `json:"-"`
	RefCount          int                           `json:"-"`
}
