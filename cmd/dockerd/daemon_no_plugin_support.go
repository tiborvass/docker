// +build !experimental !linux

package main

import (
	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/libcontainerd"
	"github.com/tiborvass/docker/registry"
)

func pluginInit(config *daemon.Config, remote libcontainerd.Remote, rs registry.Service) error {
	return nil
}
