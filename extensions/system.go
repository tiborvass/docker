package extensions

import c "github.com/docker/docker/core"

type ExtensionController interface {
	// Available returns the identifiers of available extensions
	Available() ([]c.DID, error)

	// Enabled returns the identifiers of enabled extensions
	Enabled() ([]c.DID, error)

	// Disabled returns the identifiers of disabled extensions
	Disabled([]c.DID, error)

	Get(id c.DID) (Extension, error)

	// Enable enables the specified extension, allowing it to interact with
	// the core and hook into its lifecycle.
	Enable(id c.DID) error

	// Disable disables the specified extension, removing its hooks from
	// the core lifecycle.
	Disable(id c.DID) error
}

// An extension is a object which can extend the capabilities of Docker by
// hooking into various points of its lifecycle: networking, storage, sandboxing,
// logging etc.
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
