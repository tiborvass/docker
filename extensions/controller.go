package extensions

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/core"
	"github.com/docker/docker/network"
	"github.com/docker/docker/sandbox"
	"github.com/docker/docker/state"
)

// ExtensionController manages the lifetime of loaded extensions. It provides
// an implementation of core.Core for extensions to interact with Docker.
func NewController(state state.State) *Controller {
	return &Controller{
		networks:   network.NewController(state),
		sandboxes:  sandbox.NewController(),
		extensions: make(map[core.DID]*extensionData),
	}
}

type Controller struct {
	networks   *network.Controller
	sandboxes  *sandbox.Controller
	extensions map[core.DID]*extensionData
}

type extensionData struct {
	enabled   bool
	extension Extension
}

func (c *Controller) Restore(state state.State) error {
	// Restore sandboxes first, because networking relies on their existence in
	// order to restore endpoints.
	if err := c.sandboxes.Restore(state); err != nil {
		return err
	}

	// Restore networks and endpoints.
	if err := c.networks.Restore(state); err != nil {
		return err
	}

	return nil
}

func (c *Controller) Install(id core.DID, extension Extension) error {
	if _, err := c.Get(id); err == nil {
		return fmt.Errorf("failed to install extension with duplicated ID %q", id)
	}

	extCore := newCoreProvider(c)
	if err := extension.Install(extCore); err != nil {
		return fmt.Errorf("failed to install extension: %v", err)
	}

	c.extensions[id] = &extensionData{extension: extension}
	return nil
}

func (c *Controller) Get(id core.DID) (Extension, error) {
	if ext, err := c.getExtensionData(id); err != nil {
		return nil, err
	} else {
		return ext.extension, nil
	}
}

func (c *Controller) Enable(id core.DID) error {
	ext, err := c.getExtensionData(id)
	if err != nil {
		return err
	}

	// Silently ignore is extension is already enabled.
	if ext.enabled {
		log.Debugf("Attempt to Enable() an already enabled extension %q", id)
		return nil
	}

	extCore := newCoreProvider(c)
	if err := ext.extension.Enable(extCore); err != nil {
		return err
	}

	ext.enabled = true
	return nil
}

func (c *Controller) Disable(id core.DID) error {
	ext, err := c.getExtensionData(id)
	if err != nil {
		return err
	}

	// Silently ignore is extension is already disabled.
	if !ext.enabled {
		log.Debugf("Attempt to Disable() an already disabled extension %q", id)
		return nil
	}

	extCore := newCoreProvider(c)
	if err := ext.extension.Disable(extCore); err != nil {
		return err
	}

	ext.enabled = false
	return nil
}

func (c *Controller) Available() []core.DID {
	return c.listExtensions(func(e *extensionData) bool { return true })
}

func (c *Controller) Enabled() []core.DID {
	return c.listExtensions(func(e *extensionData) bool { return e.enabled })
}

func (c *Controller) Disabled() []core.DID {
	return c.listExtensions(func(e *extensionData) bool { return !e.enabled })
}

func (c *Controller) Networks() *network.Controller {
	return c.networks
}

func (c *Controller) Sandboxes() *sandbox.Controller {
	return c.sandboxes
}

func (c *Controller) getExtensionData(id core.DID) (*extensionData, error) {
	if ext, ok := c.extensions[id]; ok {
		return ext, nil
	}
	return nil, fmt.Errorf("unknown extension ID %q", id)
}

func (c *Controller) listExtensions(predicate func(*extensionData) bool) []core.DID {
	result := make([]core.DID, 0, len(c.extensions))
	for did := range c.extensions {
		result = append(result, did)
	}
	return result
}

// The coreProvider implements extensions.Core. We rely on this extra structure
// to avoid publicly exposing Core interface in extensions.Controller.
type coreProvider struct {
	controller *Controller
}

func newCoreProvider(c *Controller) coreProvider {
	return coreProvider{controller: c}
}

func (c coreProvider) RegisterNetworkDriver(driver network.Driver, name string) error {
	// FIXME:networking Quick & dirty test code
	c.controller.networks.AddDriver(driver)
	return nil
}

func (c coreProvider) UnregisterNetworkDriver(name string) error {
	// FIXME:networking
	return nil
}
