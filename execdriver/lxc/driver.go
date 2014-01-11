package lxc

import (
	"errors"
	"fmt"
	"github.com/dotcloud/docker/execdriver"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	startPath = "lxc-start"
)

var (
	ErrNotRunning         = errors.New("Process could not be started")
	ErrWaitTimeoutReached = errors.New("Wait timeout reached")
)

type driver struct {
	root     string // root path for the driver to use
	apparmor bool
}

func NewDriver(root string, apparmor bool) (execdriver.Driver, error) {
	// setup unconfined symlink
	if err := linkLxcStart(root); err != nil {
		return nil, err
	}
	return &driver{
		apparmor: apparmor,
		root:     root,
	}, nil
}

func (d *driver) Start(c *execdriver.Process) error {
	params := []string{
		startPath,
		"-n", c.ID,
		"-f", c.ConfigPath,
		"--",
		c.InitPath,
	}

	if c.Network != nil {
		params = append(params,
			"-g", c.Network.Gateway,
			"-i", fmt.Sprintf("%s/%d", c.Network.IPAddress, c.Network.IPPrefixLen),
			"-mtu", strconv.Itoa(c.Network.Mtu),
		)
	}

	if c.User != "" {
		params = append(params, "-u", c.User)
	}

	if c.Privileged {
		if d.apparmor {
			params[0] = path.Join(d.root, "lxc-start-unconfined")

		}
		params = append(params, "-privileged")
	}

	if c.WorkingDir != "" {
		params = append(params, "-w", c.WorkingDir)
	}

	params = append(params, "--", c.Entrypoint)
	params = append(params, c.Arguments...)

	var (
		name = params[0]
		arg  = params[1:]
	)
	aname, err := exec.LookPath(name)
	if err != nil {
		aname = name
	}
	c.Path = aname
	c.Args = append([]string{name}, arg...)

	if err := c.Start(); err != nil {
		return err
	}

	go func() {
		if err := c.Wait(); err != nil {
			c.WaitError = err
		}
		close(c.WaitLock)
	}()

	// Poll for running
	if err := d.waitForStart(c); err != nil {
		return err
	}
	return nil
}

func (d *driver) Kill(c *execdriver.Process, sig int) error {
	return d.kill(c, sig)
}

func (d *driver) Wait(id string, duration time.Duration) error {
	var (
		killer bool
		done   = d.waitLxc(id, &killer)
	)

	if duration > 0 {
		select {
		case err := <-done:
			return err
		case <-time.After(duration):
			killer = true
			return ErrWaitTimeoutReached
		}
	} else {
		return <-done
	}
	return nil
}

func (d *driver) Version() string {
	version := ""
	if output, err := exec.Command("lxc-version").CombinedOutput(); err == nil {
		outputStr := string(output)
		if len(strings.SplitN(outputStr, ":", 2)) == 2 {
			version = strings.TrimSpace(strings.SplitN(outputStr, ":", 2)[1])
		}
	}
	return version
}

func (d *driver) kill(c *execdriver.Process, sig int) error {
	output, err := exec.Command("lxc-kill", "-n", c.ID, strconv.Itoa(sig)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Err: %s Output: %s", err, output)
	}
	return nil
}

func (d *driver) waitForStart(c *execdriver.Process) error {
	var (
		err    error
		output []byte
	)
	// We wait for the container to be fully running.
	// Timeout after 5 seconds. In case of broken pipe, just retry.
	// Note: The container can run and finish correctly before
	// the end of this loop
	for now := time.Now(); time.Since(now) < 5*time.Second; {
		select {
		case <-c.WaitLock:
			// If the process dies while waiting for it, just return
			if c.ProcessState != nil && c.ProcessState.Exited() {
				return nil
			}
		default:
		}

		output, err = d.getInfo(c)
		if err != nil {
			output, err = d.getInfo(c)
			if err != nil {
				return err
			}
		}
		if strings.Contains(string(output), "RUNNING") {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return ErrNotRunning
}

func (d *driver) waitLxc(id string, kill *bool) <-chan error {
	done := make(chan error)
	go func() {
		for *kill {
			output, err := exec.Command("lxc-info", "-n", id).CombinedOutput()
			if err != nil {
				done <- err
				return
			}
			if !strings.Contains(string(output), "RUNNING") {
				done <- err
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	return done
}

func (d *driver) getInfo(c *execdriver.Process) ([]byte, error) {
	return exec.Command("lxc-info", "-s", "-n", c.ID).CombinedOutput()
}

func linkLxcStart(root string) error {
	sourcePath, err := exec.LookPath("lxc-start")
	if err != nil {
		return err
	}
	targetPath := path.Join(root, "lxc-start-unconfined")

	if _, err := os.Lstat(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		if err := os.Remove(targetPath); err != nil {
			return err
		}
	}
	return os.Symlink(sourcePath, targetPath)
}
