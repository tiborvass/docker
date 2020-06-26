// +build windows

package binfmt_misc

import (
	"errors"
	"os/exec"
)

func withChroot(cmd *exec.Cmd, dir string) {
}

func check(bin string) error {
	return errors.New("binfmt is not supported on Windows")
}
