package extensions

import (
	"fmt"

	"github.com/docker/docker/extensions/context"
	"github.com/docker/docker/network"
	"github.com/docker/docker/state"
)

type ExtensionBuilder func() Extension

// The table of all "compiled-in" extensions: their names are hardcoded, and
// name resolution is achieved with a simple map lookup.
var builtinExtensions = map[string]ExtensionBuilder{}

func RegisterBuiltin(name string, builder ExtensionBuilder) error {
	if _, ok := builtinExtensions[name]; ok {
		return fmt.Errorf("builtin extension %q is already registered", name)
	}

	builtinExtensions[name] = builder
	return nil
}

func newBuiltinExtension(name string, controller *Controller, state state.State) (Extension, error) {
	extBuilder, ok := builtinExtensions[name]
	if !ok {
		return nil, fmt.Errorf("unknown builtin extension %q", name)
	}

	return &builtinExtension{
		controller: controller,
		extension:  extBuilder(),
		state:      state,
	}, nil
}

type builtinExtension struct {
	controller *Controller
	extension  Extension
	state      state.State
}

func (e *builtinExtension) newContext(parentCtx context.Context) (context.Context, error) {
	scope, err := e.state.Scope("/config")
	if err != nil {
		return nil, err
	}

	ctx := &builtinContext{
		Context:    parentCtx,
		controller: e.controller,
		extension:  e.extension,
		state:      e.state,
		config:     scope,
	}

	return ctx, nil
}

func (e *builtinExtension) Install(parentCtx context.Context) error {
	return e.dispatchWithContext(parentCtx, e.extension.Install)
}

func (e *builtinExtension) Uninstall(parentCtx context.Context) error {
	return e.dispatchWithContext(parentCtx, e.extension.Uninstall)
}

func (e *builtinExtension) Disable(parentCtx context.Context) error {
	return e.dispatchWithContext(parentCtx, e.extension.Disable)
}

func (e *builtinExtension) Enable(parentCtx context.Context) error {
	return e.dispatchWithContext(parentCtx, e.extension.Enable)
}

func (e *builtinExtension) dispatchWithContext(parentCtx context.Context, fn func(context.Context) error) error {
	if scopedCtx, err := e.newContext(parentCtx); err != nil {
		return err
	} else {
		return fn(scopedCtx)
	}
}

// builtinContext exposes the core-facing side of a Context.
type builtinContext struct {
	context.Context
	extension  Extension
	controller *Controller

	state  state.State
	config state.State
}

func (ctx *builtinContext) MyState() state.State {
	return ctx.state
}

func (ctx *builtinContext) MyConfig() state.State {
	return ctx.config
}

func (ctx *builtinContext) RegisterNetworkDriver(driver network.Driver, name string) error {
	return ctx.controller.core.RegisterNetworkDriver(driver, name)
}

func (ctx *builtinContext) UnregisterNetworkDriver(name string) error {
	return ctx.controller.core.UnregisterNetworkDriver(name)
}
