// +build !daemon

package main

import (
	"github.com/tiborvass/docker/pkg/log"
)

const CanDaemon = false

func mainDaemon() {
	log.Fatal("This is a client-only binary - running the Docker daemon is not supported.")
}
