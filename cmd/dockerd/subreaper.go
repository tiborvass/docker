package main

import (
	"fmt"
	"os"
	"github.com/opencontainers/runc/libcontainer/system"
)

func setsubreaper() {
	if os.Getenv("SUBREAPER") != "" {
		fmt.Println("subreaper error", system.SetSubreaper(1))
	}
}
