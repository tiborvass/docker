package extensions

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/extensions/context"
	"github.com/docker/docker/state"
)

// ExtensionController manages the lifetime of loaded extensions.
func NewController(state state.State, core context.Core) *Controller {
	return &Controller{
		core:       core,
		extensions: make(map[string]*extensionData),
		state:      state,
	}
}

// NOTE netdriver: all extensions are initialized by calling New...()
// and passing a dedicated state object.
// The extension is responsible for 1) reading initial state for initialization,
// and 2) continue watching for state changes to resolve them. This means
// extensions need a way to spawn long-running goroutines. The core is
// responsible for providing a facility for that.
type Controller struct {
	core       context.Core
	extensions map[string]*extensionData
	state      state.State
}

type extensionData struct {
	enabled   bool
	extension Extension
}

func (c *Controller) Restore(state state.State) error {
	// Go over all extensions.
	// Re-initialize those that are activated.
	return nil
}

// Install attempts to find the extension matching the specified names and
// initializes it. Once installed, it can respond to get, enable, and disable
// messages.
func (c *Controller) Install(name string) error {
	ext, err := c.load(name)
	if err == nil {
		return fmt.Errorf("failed to install extension with duplicated name %q", name)
	}

	ctx := context.Root()
	if err := ext.Install(ctx); err != nil {
		return fmt.Errorf("failed to install extension: %v", err)
	}

	c.extensions[name] = &extensionData{extension: ext}
	return nil
}

// Load returns the extension corresponding to the specified name, or an error
// if either no extension was found, or an extension with that name is already
// registered.
func (c *Controller) load(name string) (Extension, error) {
	if _, ok := c.extensions[name]; ok {
		return nil, fmt.Errorf("extention %q is already loaded", name)
	}

	state, err := c.state.Scope("extensions/" + name + "/state")
	if err != nil {
		return nil, err
	}

	return newBuiltinExtension(name, c, state)
}

// Get returns an extension previously installed under the specified name.
func (c *Controller) Get(name string) (Extension, error) {
	if extData, ok := c.extensions[name]; ok {
		return extData.extension, nil
	}
	return nil, fmt.Errorf("extension %q is not installed", name)
}

func (c *Controller) Enable(name string) error {
	ext, err := c.getExtensionData(name)
	if err != nil {
		return err
	}

	// Silently ignore is extension is already enabled.
	if ext.enabled {
		log.Debugf("Attempt to Enable() an already enabled extension %q", name)
		return nil
	}

	ctx := context.Root()
	if err := ext.extension.Enable(ctx); err != nil {
		return err
	}

	ext.enabled = true
	return nil
}

func (c *Controller) Disable(name string) error {
	ext, err := c.getExtensionData(name)
	if err != nil {
		return err
	}

	// Silently ignore is extension is already disabled.
	if !ext.enabled {
		log.Debugf("Attempt to Disable() an already disabled extension %q", name)
		return nil
	}

	ctx := context.Root()
	if err := ext.extension.Disable(ctx); err != nil {
		return err
	}

	ext.enabled = false
	return nil
}

func (c *Controller) Available() []string {
	return c.listExtensions(func(e *extensionData) bool { return true })
}

func (c *Controller) Enabled() []string {
	return c.listExtensions(func(e *extensionData) bool { return e.enabled })
}

func (c *Controller) Disabled() []string {
	return c.listExtensions(func(e *extensionData) bool { return !e.enabled })
}

func (c *Controller) getExtensionData(name string) (*extensionData, error) {
	if ext, ok := c.extensions[name]; ok {
		return ext, nil
	}
	return nil, fmt.Errorf("unknown extension %q", name)
}

func (c *Controller) listExtensions(predicate func(*extensionData) bool) []string {
	result := make([]string, 0, len(c.extensions))
	for name := range c.extensions {
		result = append(result, name)
	}
	return result
}
