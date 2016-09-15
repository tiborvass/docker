package store

import "github.com/tiborvass/docker/pkg/plugins"

const (
	// LOOKUP doesn't update RefCount
	LOOKUP = 0
	// CREATE increments RefCount
	CREATE = 1
	// REMOVE decrements RefCount
	REMOVE = -1
)

// CompatPlugin is an abstraction to handle both new and legacy (v1) plugins.
type CompatPlugin interface {
	Client() *plugins.Client
	Name() string
	IsLegacy() bool
}
