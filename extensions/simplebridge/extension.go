package simplebridge

import (
	"github.com/docker/docker/extensions"
)

type Extension struct{}

func (e Extension) Install(c extensions.Context) error   { return nil }
func (e Extension) Uninstall(c extensions.Context) error { return nil }
func (e Extension) Enable(c extensions.Context) error    { return nil }
func (e Extension) Disable(c extensions.Context) error   { return nil }
