// +build !daemon

package main

import "github.com/tiborvass/docker/cli"

const daemonUsage = ""

var daemonCli cli.Handler

// TODO: remove once `-d` is retired
func handleGlobalDaemonFlag() {}
