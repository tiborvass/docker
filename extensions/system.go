package extensions

import "github.com/docker/docker/network"

type Core interface {
	RegisterNetworkDriver(driver network.Driver, name string) error
	UnregisterNetworkDriver(name string) error
}

// An extension is a object which can extend the capabilities of Docker by
// hooking into various points of its lifecycle: networking, storage, sandboxing,
// logging etcore.
type Extension interface {
	// Install is called when the extension is first installed.
	// The extension should use it for one-time initialization of resources which
	// it will need later.
	//
	// Once installed the extension must be enabled separately. Install MUST NOT
	// interfere with the functioning and user experience of Docker.
	//
	Install(c Core) error

	// Uninstall is called when the extension is uninstalled.
	// The extension should use it to tear down resources initialized at install,
	// and cleaning up the host environment of any side effects.
	Uninstall(c Core) error

	// Enable is called when a) the user enables the extension, or b) the daemon is starting
	// and the extension is already enabled.
	//
	// The extension should use it to hook itself into the core to modify its behavior.
	// See the Core interface for available interactions with the core.
	//
	Enable(c Core) error

	// Disabled is called when the extension is disabled.
	Disable(c Core) error
}
