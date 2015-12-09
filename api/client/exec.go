package client

import (
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api/types"
	Cli "github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/pkg/promise"
	"github.com/tiborvass/docker/runconfig"
)

// CmdExec runs a command in a running container.
//
// Usage: docker exec [OPTIONS] CONTAINER COMMAND [ARG...]
func (cli *DockerCli) CmdExec(args ...string) error {
	cmd := Cli.Subcmd("exec", []string{"CONTAINER COMMAND [ARG...]"}, Cli.DockerCommands["exec"].Description, true)

	execConfig, err := runconfig.ParseExec(cmd, args)
	// just in case the ParseExec does not exit
	if execConfig.Container == "" || err != nil {
		return Cli.StatusError{StatusCode: 1}
	}

	response, err := cli.client.ContainerExecCreate(*execConfig)
	if err != nil {
		return err
	}

	execID := response.ID
	if execID == "" {
		fmt.Fprintf(cli.out, "exec ID empty")
		return nil
	}

	//Temp struct for execStart so that we don't need to transfer all the execConfig
	if !execConfig.Detach {
		if err := cli.CheckTtyInput(execConfig.AttachStdin, execConfig.Tty); err != nil {
			return err
		}
	} else {
		execStartCheck := types.ExecStartCheck{
			Detach: execConfig.Detach,
			Tty:    execConfig.Tty,
		}

		if err := cli.client.ContainerExecStart(execID, execStartCheck); err != nil {
			return err
		}
		// For now don't print this - wait for when we support exec wait()
		// fmt.Fprintf(cli.out, "%s\n", execID)
		return nil
	}

	// Interactive exec requested.
	var (
		out, stderr io.Writer
		in          io.ReadCloser
		errCh       chan error
	)

	if execConfig.AttachStdin {
		in = cli.in
	}
	if execConfig.AttachStdout {
		out = cli.out
	}
	if execConfig.AttachStderr {
		if execConfig.Tty {
			stderr = cli.out
		} else {
			stderr = cli.err
		}
	}

	resp, err := cli.client.ContainerExecAttach(execID, *execConfig)
	if err != nil {
		return err
	}
	defer resp.Close()
	errCh = promise.Go(func() error {
		return cli.holdHijackedConnection(execConfig.Tty, in, out, stderr, resp)
	})

	if execConfig.Tty && cli.isTerminalIn {
		if err := cli.monitorTtySize(execID, true); err != nil {
			fmt.Fprintf(cli.err, "Error monitoring TTY size: %s\n", err)
		}
	}

	if err := <-errCh; err != nil {
		logrus.Debugf("Error hijack: %s", err)
		return err
	}

	var status int
	if _, status, err = getExecExitCode(cli, execID); err != nil {
		return err
	}

	if status != 0 {
		return Cli.StatusError{StatusCode: status}
	}

	return nil
}
