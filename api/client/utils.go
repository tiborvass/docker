package client

import (
	"fmt"
	"os"
	gosignal "os/signal"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api/client/lib"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/pkg/signal"
	"github.com/tiborvass/docker/pkg/term"
	"github.com/tiborvass/docker/registry"
)

func (cli *DockerCli) encodeRegistryAuth(index *registry.IndexInfo) (string, error) {
	authConfig := registry.ResolveAuthConfig(cli.configFile, index)
	return authConfig.EncodeToBase64()
}

func (cli *DockerCli) registryAuthenticationPrivilegedFunc(index *registry.IndexInfo, cmdName string) lib.RequestPrivilegeFunc {
	return func() (string, error) {
		fmt.Fprintf(cli.out, "\nPlease login prior to %s:\n", cmdName)
		if err := cli.CmdLogin(index.GetAuthConfigKey()); err != nil {
			return "", err
		}
		return cli.encodeRegistryAuth(index)
	}
}

func (cli *DockerCli) resizeTty(id string, isExec bool) {
	height, width := cli.getTtySize()
	if height == 0 && width == 0 {
		return
	}

	options := types.ResizeOptions{
		ID:     id,
		Height: height,
		Width:  width,
	}

	var err error
	if !isExec {
		err = cli.client.ContainerExecResize(options)
	} else {
		err = cli.client.ContainerResize(options)
	}

	if err != nil {
		logrus.Debugf("Error resize: %s", err)
	}
}

// getExitCode perform an inspect on the container. It returns
// the running state and the exit code.
func getExitCode(cli *DockerCli, containerID string) (bool, int, error) {
	c, err := cli.client.ContainerInspect(containerID)
	if err != nil {
		// If we can't connect, then the daemon probably died.
		if err != lib.ErrConnectionFailed {
			return false, -1, err
		}
		return false, -1, nil
	}

	return c.State.Running, c.State.ExitCode, nil
}

// getExecExitCode perform an inspect on the exec command. It returns
// the running state and the exit code.
func getExecExitCode(cli *DockerCli, execID string) (bool, int, error) {
	resp, err := cli.client.ContainerExecInspect(execID)
	if err != nil {
		// If we can't connect, then the daemon probably died.
		if err != lib.ErrConnectionFailed {
			return false, -1, err
		}
		return false, -1, nil
	}

	return resp.Running, resp.ExitCode, nil
}

func (cli *DockerCli) monitorTtySize(id string, isExec bool) error {
	cli.resizeTty(id, isExec)

	if runtime.GOOS == "windows" {
		go func() {
			prevH, prevW := cli.getTtySize()
			for {
				time.Sleep(time.Millisecond * 250)
				h, w := cli.getTtySize()

				if prevW != w || prevH != h {
					cli.resizeTty(id, isExec)
				}
				prevH = h
				prevW = w
			}
		}()
	} else {
		sigchan := make(chan os.Signal, 1)
		gosignal.Notify(sigchan, signal.SIGWINCH)
		go func() {
			for range sigchan {
				cli.resizeTty(id, isExec)
			}
		}()
	}
	return nil
}

func (cli *DockerCli) getTtySize() (int, int) {
	if !cli.isTerminalOut {
		return 0, 0
	}
	ws, err := term.GetWinsize(cli.outFd)
	if err != nil {
		logrus.Debugf("Error getting size: %s", err)
		if ws == nil {
			return 0, 0
		}
	}
	return int(ws.Height), int(ws.Width)
}
