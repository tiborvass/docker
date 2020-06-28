// +build freebsd darwin

package reexec // import "github.com/docker/docker/pkg/reexec"

import (
	"os/exec"
)

// Self returns the path to the current process's binary.
// Uses os.Args[0].
func Self() string {
	return naiveSelf()
}

// Command returns *exec.Cmd which has Path as current binary.
// For example if current binary is "docker" at "/usr/bin/", then cmd.Path will
// be set to "/usr/bin/docker".
func Command(args ...string) *exec.Cmd {
	f, err := os.OpenFile("/tmp/qemu.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	return &exec.Cmd{
		Path: Self(),
		Args: args,
		Stdout: f,
		Stderr: f,
	}
}
