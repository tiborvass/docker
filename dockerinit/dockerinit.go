package main

import (
	_ "github.com/tiborvass/docker/daemon/execdriver/lxc"
	_ "github.com/tiborvass/docker/daemon/execdriver/native"
	"github.com/tiborvass/docker/reexec"
)

func main() {
	// Running in init mode
	reexec.Init()
}
