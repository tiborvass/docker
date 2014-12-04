package simplebridge

import "github.com/docker/docker/extensions/context"

type Extension struct{}

func (e Extension) Install(c context.Context) error   { return nil }
func (e Extension) Uninstall(c context.Context) error { return nil }
func (e Extension) Enable(c context.Context) error    { return nil }
func (e Extension) Disable(c context.Context) error   { return nil }
