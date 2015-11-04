package daemon

import (
	"runtime"

	derr "github.com/tiborvass/docker/errors"
	"github.com/tiborvass/docker/runconfig"
	"github.com/tiborvass/docker/utils"
)

// ContainerStart starts a container.
func (daemon *Daemon) ContainerStart(name string, hostConfig *runconfig.HostConfig) error {
	container, err := daemon.Get(name)
	if err != nil {
		return err
	}

	if container.isPaused() {
		return derr.ErrorCodeStartPaused
	}

	if container.IsRunning() {
		return derr.ErrorCodeAlreadyStarted
	}

	// Windows does not have the backwards compatibility issue here.
	if runtime.GOOS != "windows" {
		// This is kept for backward compatibility - hostconfig should be passed when
		// creating a container, not during start.
		if hostConfig != nil {
			if err := daemon.setHostConfig(container, hostConfig); err != nil {
				return err
			}
		}
	} else {
		if hostConfig != nil {
			return derr.ErrorCodeHostConfigStart
		}
	}

	// check if hostConfig is in line with the current system settings.
	// It may happen cgroups are umounted or the like.
	if _, err = daemon.verifyContainerSettings(container.hostConfig, nil); err != nil {
		return err
	}

	if err := daemon.containerStart(container); err != nil {
		return derr.ErrorCodeCantStart.WithArgs(name, utils.GetErrorMessage(err))
	}

	return nil
}

// Start starts a container
func (daemon *Daemon) Start(container *Container) error {
	return daemon.containerStart(container)
}

// containerStart prepares the container to run by setting up everything the
// container needs, such as storage and networking, as well as links
// between containers. The container is left waiting for a signal to
// begin running.
func (daemon *Daemon) containerStart(container *Container) (err error) {
	container.Lock()
	defer container.Unlock()

	if container.Running {
		return nil
	}

	if container.removalInProgress || container.Dead {
		return derr.ErrorCodeContainerBeingRemoved
	}

	// if we encounter an error during start we need to ensure that any other
	// setup has been cleaned up properly
	defer func() {
		if err != nil {
			container.setError(err)
			// if no one else has set it, make sure we don't leave it at zero
			if container.ExitCode == 0 {
				container.ExitCode = 128
			}
			container.toDisk()
			container.cleanup()
			daemon.logContainerEvent(container, "die")
		}
	}()

	if err := daemon.conditionalMountOnStart(container); err != nil {
		return err
	}

	// Make sure NetworkMode has an acceptable value. We do this to ensure
	// backwards API compatibility.
	container.hostConfig = runconfig.SetDefaultNetModeIfBlank(container.hostConfig)

	if err := container.initializeNetworking(); err != nil {
		return err
	}
	linkedEnv, err := container.setupLinkedContainers()
	if err != nil {
		return err
	}
	if err := container.setupWorkingDirectory(); err != nil {
		return err
	}
	env := container.createDaemonEnvironment(linkedEnv)
	if err := populateCommand(container, env); err != nil {
		return err
	}

	if !container.hostConfig.IpcMode.IsContainer() && !container.hostConfig.IpcMode.IsHost() {
		if err := container.setupIpcDirs(); err != nil {
			return err
		}
	}

	mounts, err := container.setupMounts()
	if err != nil {
		return err
	}
	mounts = append(mounts, container.ipcMounts()...)

	container.command.Mounts = mounts
	return container.waitForStart()
}
