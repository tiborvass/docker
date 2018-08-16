package supervisor // import "github.com/tiborvass/docker/libcontainerd/supervisor"

import "syscall"

// containerdSysProcAttr returns the SysProcAttr to use when exec'ing
// containerd
func containerdSysProcAttr() *syscall.SysProcAttr {
	return nil
}
