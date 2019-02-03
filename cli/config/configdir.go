package config // import "github.com/tiborvass/docker/cli/config"

import (
	"os"
	"path/filepath"

	"github.com/tiborvass/docker/pkg/homedir"
)

var (
	configDir     = os.Getenv("DOCKER_CONFIG")
	configFileDir = ".docker"
)

// Dir returns the path to the configuration directory as specified by the DOCKER_CONFIG environment variable.
// If DOCKER_CONFIG is unset, Dir returns ~/.docker .
// Dir ignores XDG_CONFIG_HOME (same as the docker client).
// TODO: this was copied from cli/config/configfile and should be removed once cmd/dockerd moves
func Dir() string {
	return configDir
}

func init() {
	if configDir == "" {
		configDir = filepath.Join(homedir.Get(), configFileDir)
	}
}
