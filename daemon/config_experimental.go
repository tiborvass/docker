// +build experimental

package daemon

import flag "github.com/tiborvass/docker/pkg/mflag"

func (config *Config) attachExperimentalFlags(cmd *flag.FlagSet, usageFn func(string) string) {
	cmd.StringVar(&config.DefaultNetwork, []string{"-default-network"}, "", usageFn("Set default network"))
}
