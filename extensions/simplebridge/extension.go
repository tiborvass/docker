package simplebridge

import (
	"github.com/docker/docker/extensions"
)

type Extension struct{}

func (e Extension) Install(c extensions.Core) error   { return nil }
func (e Extension) Uninstall(c extensions.Core) error { return nil }
func (e Extension) Enable(c extensions.Core) error    { return nil }
func (e Extension) Disable(c extensions.Core) error   { return nil }
