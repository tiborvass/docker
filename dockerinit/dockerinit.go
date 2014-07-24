package main

import (
	"github.com/tiborvass/docker/sysinit"
)

func main() {
	// Running in init mode
	sysinit.SysInit()
	return
}
