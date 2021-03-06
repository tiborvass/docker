// +build !windows

package main

import (
	"bytes"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/tiborvass/docker/pkg/parsers/kernel"
	"github.com/tiborvass/docker/pkg/sysinfo"
)

var (
	// SysInfo stores information about which features a kernel supports.
	SysInfo *sysinfo.SysInfo
)

func cpuCfsPeriod() bool {
	return testEnv.DaemonInfo.CPUCfsPeriod
}

func cpuCfsQuota() bool {
	return testEnv.DaemonInfo.CPUCfsQuota
}

func cpuShare() bool {
	return testEnv.DaemonInfo.CPUShares
}

func oomControl() bool {
	return testEnv.DaemonInfo.OomKillDisable
}

func pidsLimit() bool {
	return SysInfo.PidsLimit
}

func kernelMemorySupport() bool {
	// TODO remove this once kmem support in RHEL kernels is fixed. See https://github.com/opencontainers/runc/pull/1921
	daemonV, err := kernel.ParseRelease(testEnv.DaemonInfo.KernelVersion)
	if err != nil {
		return false
	}
	requiredV := kernel.VersionInfo{Kernel: 3, Major: 10}
	if kernel.CompareKernelVersion(*daemonV, requiredV) < 1 {
		// On Kernel 3.10 and under, don't consider kernel memory to be supported,
		// even if the kernel (and thus the daemon) reports it as being supported
		return false
	}
	return testEnv.DaemonInfo.KernelMemory
}

func memoryLimitSupport() bool {
	return testEnv.DaemonInfo.MemoryLimit
}

func memoryReservationSupport() bool {
	return SysInfo.MemoryReservation
}

func swapMemorySupport() bool {
	return testEnv.DaemonInfo.SwapLimit
}

func memorySwappinessSupport() bool {
	return testEnv.IsLocalDaemon() && SysInfo.MemorySwappiness
}

func blkioWeight() bool {
	return testEnv.IsLocalDaemon() && SysInfo.BlkioWeight
}

func cgroupCpuset() bool {
	return testEnv.DaemonInfo.CPUSet
}

func seccompEnabled() bool {
	return supportsSeccomp && SysInfo.Seccomp
}

func bridgeNfIptables() bool {
	return !SysInfo.BridgeNFCallIPTablesDisabled
}

func unprivilegedUsernsClone() bool {
	content, err := ioutil.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	return err != nil || !strings.Contains(string(content), "0")
}

func overlayFSSupported() bool {
	cmd := exec.Command(dockerBinary, "run", "--rm", "busybox", "/bin/sh", "-c", "cat /proc/filesystems")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return bytes.Contains(out, []byte("overlay\n"))
}

func init() {
	if testEnv.IsLocalDaemon() {
		SysInfo = sysinfo.New(true)
	}
}
