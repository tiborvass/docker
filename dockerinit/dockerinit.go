package main

import (
	_ "github.com/tiborvass/docker/daemon/execdriver/lxc"
	_ "github.com/tiborvass/docker/daemon/execdriver/native"
	"github.com/tiborvass/docker/pkg/reexec"
)

func main() {
	// Running in init mode
	reexec.Init()
}
