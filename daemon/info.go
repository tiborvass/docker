package daemon

import (
	"os"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/dockerversion"
	"github.com/tiborvass/docker/pkg/fileutils"
	"github.com/tiborvass/docker/pkg/parsers/kernel"
	"github.com/tiborvass/docker/pkg/parsers/operatingsystem"
	"github.com/tiborvass/docker/pkg/platform"
	"github.com/tiborvass/docker/pkg/sysinfo"
	"github.com/tiborvass/docker/pkg/system"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/utils"
	"github.com/tiborvass/docker/volume/drivers"
)

// SystemInfo returns information about the host server the daemon is running on.
func (daemon *Daemon) SystemInfo() (*types.Info, error) {
	kernelVersion := "<unknown>"
	if kv, err := kernel.GetKernelVersion(); err == nil {
		kernelVersion = kv.String()
	}

	operatingSystem := "<unknown>"
	if s, err := operatingsystem.GetOperatingSystem(); err == nil {
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
	}

	// if we still have the original dockerinit binary from before
	// we copied it locally, let's return the path to that, since
	// that's more intuitive (the copied path is trivial to derive
	// by hand given VERSION)
	initPath := utils.DockerInitPath("")
	if initPath == "" {
		// if that fails, we'll just return the path from the daemon
		initPath = daemon.systemInitPath()
	}

	sysInfo := sysinfo.New(true)

	v := &types.Info{
		ID:                 daemon.ID,
		Containers:         len(daemon.List()),
		Images:             len(daemon.Graph().Map()),
		Driver:             daemon.GraphDriver().String(),
		DriverStatus:       daemon.GraphDriver().Status(),
		Plugins:            daemon.showPluginsInfo(),
		IPv4Forwarding:     !sysInfo.IPv4ForwardingDisabled,
		BridgeNfIptables:   !sysInfo.BridgeNfCallIptablesDisabled,
		BridgeNfIP6tables:  !sysInfo.BridgeNfCallIP6tablesDisabled,
		Debug:              os.Getenv("DEBUG") != "",
		NFd:                fileutils.GetTotalUsedFds(),
		NGoroutines:        runtime.NumGoroutine(),
		SystemTime:         time.Now().Format(time.RFC3339Nano),
		ExecutionDriver:    daemon.ExecutionDriver().Name(),
		LoggingDriver:      daemon.defaultLogConfig.Type,
		NEventsListener:    daemon.EventsService.SubscribersCount(),
		KernelVersion:      kernelVersion,
		OperatingSystem:    operatingSystem,
		IndexServerAddress: registry.IndexServer,
		OSType:             platform.OSType,
		Architecture:       platform.Architecture,
		RegistryConfig:     daemon.RegistryService.Config,
		InitSha1:           dockerversion.InitSHA1,
		InitPath:           initPath,
		NCPU:               runtime.NumCPU(),
		MemTotal:           meminfo.MemTotal,
		DockerRootDir:      daemon.config().Root,
		Labels:             daemon.config().Labels,
		ExperimentalBuild:  utils.ExperimentalBuild(),
		ServerVersion:      dockerversion.Version,
		ClusterStore:       daemon.config().ClusterStore,
		ClusterAdvertise:   daemon.config().ClusterAdvertise,
		HTTPProxy:          os.Getenv("http_proxy"),
		HTTPSProxy:         os.Getenv("https_proxy"),
		NoProxy:            os.Getenv("no_proxy"),
	}

	// TODO Windows. Refactor this more once sysinfo is refactored into
	// platform specific code. On Windows, sysinfo.cgroupMemInfo and
	// sysinfo.cgroupCpuInfo will be nil otherwise and cause a SIGSEGV if
	// an attempt is made to access through them.
	if runtime.GOOS != "windows" {
		v.MemoryLimit = sysInfo.MemoryLimit
		v.SwapLimit = sysInfo.SwapLimit
		v.OomKillDisable = sysInfo.OomKillDisable
		v.CPUCfsPeriod = sysInfo.CPUCfsPeriod
		v.CPUCfsQuota = sysInfo.CPUCfsQuota
		v.CPUShares = sysInfo.CPUShares
		v.CPUSet = sysInfo.Cpuset
	}

	if hostname, err := os.Hostname(); err == nil {
		v.Name = hostname
	}

	return v, nil
}

func (daemon *Daemon) showPluginsInfo() types.PluginsInfo {
	var pluginsInfo types.PluginsInfo

	pluginsInfo.Volume = volumedrivers.GetDriverList()

	networkDriverList := daemon.GetNetworkDriverList()
	for nd := range networkDriverList {
		pluginsInfo.Network = append(pluginsInfo.Network, nd)
	}

	return pluginsInfo
}
