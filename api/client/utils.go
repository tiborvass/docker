package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	gosignal "os/signal"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/pkg/signal"
	"github.com/tiborvass/docker/pkg/term"
	"github.com/tiborvass/docker/registry"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	registrytypes "github.com/docker/engine-api/types/registry"
)

// encodeAuthToBase64 serializes the auth configuration as JSON base64 payload
func encodeAuthToBase64(authConfig types.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

func (cli *DockerCli) encodeRegistryAuth(index *registrytypes.IndexInfo) (string, error) {
	authConfig := registry.ResolveAuthConfig(cli.configFile.AuthConfigs, index)
	return encodeAuthToBase64(authConfig)
}

func (cli *DockerCli) registryAuthenticationPrivilegedFunc(index *registrytypes.IndexInfo, cmdName string) client.RequestPrivilegeFunc {
	return func() (string, error) {
		fmt.Fprintf(cli.out, "\nPlease login prior to %s:\n", cmdName)
		indexServer := registry.GetAuthConfigKey(index)
		authConfig, err := cli.configureAuth("", "", "", indexServer)
		if err != nil {
			return "", err
		}
		return encodeAuthToBase64(authConfig)
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
	if isExec {
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
		if err != client.ErrConnectionFailed {
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
		if err != client.ErrConnectionFailed {
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
