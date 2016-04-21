package main

import (
	"path/filepath"

	cliflags "github.com/tiborvass/docker/cli/flags"
	"github.com/tiborvass/docker/cliconfig"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/utils"
)

var (
	commonFlags = cliflags.InitCommonFlags()
	clientFlags = &cliflags.ClientFlags{FlagSet: new(flag.FlagSet), Common: commonFlags}
)

func init() {

	client := clientFlags.FlagSet
	client.StringVar(&clientFlags.ConfigDir, []string{"-config"}, cliconfig.ConfigDir(), "Location of client config files")

	clientFlags.PostParse = func() {
		clientFlags.Common.PostParse()

		if clientFlags.ConfigDir != "" {
			cliconfig.SetConfigDir(clientFlags.ConfigDir)
		}

		if clientFlags.Common.TrustKey == "" {
			clientFlags.Common.TrustKey = filepath.Join(cliconfig.ConfigDir(), cliflags.DefaultTrustKeyFile)
		}

		if clientFlags.Common.Debug {
			utils.EnableDebug()
		}
	}
}
