// +build daemon

package main

import (
	apiserver "github.com/tiborvass/docker/api/server"
	"github.com/tiborvass/docker/daemon"
)

func setPlatformServerConfig(serverConfig *apiserver.ServerConfig, daemonCfg *daemon.Config) *apiserver.ServerConfig {
	return serverConfig
}

// currentUserIsOwner checks whether the current user is the owner of the given
// file.
func currentUserIsOwner(f string) bool {
	return false
}
