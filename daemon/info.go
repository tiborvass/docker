package daemon

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/cli/debug"
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/dockerversion"
	"github.com/tiborvass/docker/pkg/fileutils"
	"github.com/tiborvass/docker/pkg/parsers/kernel"
	"github.com/tiborvass/docker/pkg/parsers/operatingsystem"
	"github.com/tiborvass/docker/pkg/platform"
	"github.com/tiborvass/docker/pkg/sysinfo"
	"github.com/tiborvass/docker/pkg/system"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/volume/drivers"
	"github.com/docker/go-connections/sockets"
)

// SystemInfo returns information about the host server the daemon is running on.
func (daemon *Daemon) SystemInfo() (*types.Info, error) {
	kernelVersion := "<unknown>"
	if kv, err := kernel.GetKernelVersion(); err != nil {
		logrus.Warnf("Could not get kernel version: %v", err)
	} else {
		kernelVersion = kv.String()
	}

	operatingSystem := "<unknown>"
	if s, err := operatingsystem.GetOperatingSystem(); err != nil {
		logrus.Warnf("Could not get operating system name: %v", err)
	} else {
		operatingSystem = s
	}

	// Don't do containerized check on Windows
	if runtime.GOOS != "windows" {
		if inContainer, err := operatingsystem.IsContainerized(); err != nil {
			logrus.Errorf("Could not determine if daemon is containerized: %v", err)
			operatingSystem += " (error determining if containerized)"
		} else if inContainer {
			operatingSystem += " (containerized)"
		}
	}

	meminfo, err := system.ReadMemInfo()
	if err != nil {
		logrus.Errorf("Could not read system memory info: %v", err)
		meminfo = &system.MemInfo{}
	}

	sysInfo := sysinfo.New(true)

	var cRunning, cPaused, cStopped int32
	daemon.containers.ApplyAll(func(c *container.Container) {
		switch c.StateString() {
		case "paused":
			atomic.AddInt32(&cPaused, 1)
		case "running":
			atomic.AddInt32(&cRunning, 1)
		default:
			atomic.AddInt32(&cStopped, 1)
		}
	})

	securityOptions := []string{}
	if sysInfo.AppArmor {
		securityOptions = append(securityOptions, "name=apparmor")
	}
	if sysInfo.Seccomp && supportsSeccomp {
		profile := daemon.seccompProfilePath
		if profile == "" {
			profile = "default"
		}
		securityOptions = append(securityOptions, fmt.Sprintf("name=seccomp,profile=%s", profile))
	}
	if selinuxEnabled() {
		securityOptions = append(securityOptions, "name=selinux")
	}
	uid, gid := daemon.GetRemappedUIDGID()
	if uid != 0 || gid != 0 {
		securityOptions = append(securityOptions, "name=userns")
	}

	v := &types.Info{
		ID:                 daemon.ID,
		Containers:         int(cRunning + cPaused + cStopped),
		ContainersRunning:  int(cRunning),
		ContainersPaused:   int(cPaused),
		ContainersStopped:  int(cStopped),
		Images:             len(daemon.imageStore.Map()),
		Driver:             daemon.GraphDriverName(),
		DriverStatus:       daemon.layerStore.DriverStatus(),
		Plugins:            daemon.showPluginsInfo(),
		IPv4Forwarding:     !sysInfo.IPv4ForwardingDisabled,
		BridgeNfIptables:   !sysInfo.BridgeNFCallIPTablesDisabled,
		BridgeNfIP6tables:  !sysInfo.BridgeNFCallIP6TablesDisabled,
		Debug:              debug.IsEnabled(),
		NFd:                fileutils.GetTotalUsedFds(),
		NGoroutines:        runtime.NumGoroutine(),
		SystemTime:         time.Now().Format(time.RFC3339Nano),
		LoggingDriver:      daemon.defaultLogConfig.Type,
		CgroupDriver:       daemon.getCgroupDriver(),
		NEventsListener:    daemon.EventsService.SubscribersCount(),
		KernelVersion:      kernelVersion,
		OperatingSystem:    operatingSystem,
		IndexServerAddress: registry.IndexServer,
		OSType:             platform.OSType,
		Architecture:       platform.Architecture,
		RegistryConfig:     daemon.RegistryService.ServiceConfig(),
		NCPU:               sysinfo.NumCPU(),
		MemTotal:           meminfo.MemTotal,
		DockerRootDir:      daemon.configStore.Root,
		Labels:             daemon.configStore.Labels,
		ExperimentalBuild:  daemon.configStore.Experimental,
		ServerVersion:      dockerversion.Version,
		ClusterStore:       daemon.configStore.ClusterStore,
		ClusterAdvertise:   daemon.configStore.ClusterAdvertise,
		HTTPProxy:          sockets.GetProxyEnv("http_proxy"),
		HTTPSProxy:         sockets.GetProxyEnv("https_proxy"),
		NoProxy:            sockets.GetProxyEnv("no_proxy"),
		LiveRestoreEnabled: daemon.configStore.LiveRestoreEnabled,
		SecurityOptions:    securityOptions,
		Isolation:          daemon.defaultIsolation,
	}

	// Retrieve platform specific info
	daemon.FillPlatformInfo(v, sysInfo)

	hostname := ""
	if hn, err := os.Hostname(); err != nil {
		logrus.Warnf("Could not get hostname: %v", err)
	} else {
		hostname = hn
	}
	v.Name = hostname

	return v, nil
}

// SystemVersion returns version information about the daemon.
func (daemon *Daemon) SystemVersion() types.Version {
	v := types.Version{
		Version:       dockerversion.Version,
		GitCommit:     dockerversion.GitCommit,
		MinAPIVersion: api.MinVersion,
		GoVersion:     runtime.Version(),
		Os:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		BuildTime:     dockerversion.BuildTime,
		Experimental:  daemon.configStore.Experimental,
	}

	kernelVersion := "<unknown>"
	if kv, err := kernel.GetKernelVersion(); err != nil {
		logrus.Warnf("Could not get kernel version: %v", err)
	} else {
		kernelVersion = kv.String()
	}
	v.KernelVersion = kernelVersion

	return v
}

func (daemon *Daemon) showPluginsInfo() types.PluginsInfo {
	var pluginsInfo types.PluginsInfo

	pluginsInfo.Volume = volumedrivers.GetDriverList()
	pluginsInfo.Network = daemon.GetNetworkDriverList()
	pluginsInfo.Authorization = daemon.configStore.GetAuthorizationPlugins()

	return pluginsInfo
}
