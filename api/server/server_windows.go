// +build windows

package server

import (
	"fmt"

	"github.com/tiborvass/docker/engine"
)

// NewServer sets up the required Server and does protocol specific checking.
func NewServer(proto, addr string, job *engine.Job) (Server, error) {
	// Basic error and sanity checking
	switch proto {
	case "tcp":
		return setupTcpHttp(addr, job)
	default:
		return nil, errors.New("Invalid protocol format. Windows only supports tcp.")
	}
}
