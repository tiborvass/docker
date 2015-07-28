// +build daemon,!windows

package main

import (
	"fmt"
	"os"
	"syscall"

	apiserver "github.com/tiborvass/docker/api/server"
	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/pkg/system"

	_ "github.com/tiborvass/docker/daemon/execdriver/lxc"
	_ "github.com/tiborvass/docker/daemon/execdriver/native"
)

func setPlatformServerConfig(serverConfig *apiserver.ServerConfig, daemonCfg *daemon.Config) *apiserver.ServerConfig {
	serverConfig.SocketGroup = daemonCfg.SocketGroup
	serverConfig.EnableCors = daemonCfg.EnableCors
	serverConfig.CorsHeaders = daemonCfg.CorsHeaders

	return serverConfig
}

// currentUserIsOwner checks whether the current user is the owner of the given
// file.
func currentUserIsOwner(f string) bool {
	if fileInfo, err := system.Stat(f); err == nil && fileInfo != nil {
		if int(fileInfo.Uid()) == os.Getuid() {
			return true
		}
	}
	return false
}

// setDefaultUmask sets the umask to 0022 to avoid problems
// caused by custom umask
func setDefaultUmask() error {
	desiredUmask := 0022
	syscall.Umask(desiredUmask)
	if umask := syscall.Umask(desiredUmask); umask != desiredUmask {
		return fmt.Errorf("failed to set umask: expected %#o, got %#o", desiredUmask, umask)
	}

	return nil
}
