package lxc

import (
	"errors"
	"fmt"
	"github.com/dotcloud/docker/execdriver"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	startPath = "lxc-start"
)

var (
	ErrNotRunning         = errors.New("Process could not be started")
	ErrWaitTimeoutReached = errors.New("Wait timeout reached")
)

func init() {
	// Register driver
}

type driver struct {
	root       string // root path for the driver to use
	containers map[string]*execdriver.Process
}

func NewDriver(root string) (execdriver.Driver, error) {
	// setup unconfined symlink
	return &driver{
		root:       root,
		containers: make(map[string]*execdriver.Process),
	}, nil
}

func (d *driver) Start(c *execdriver.Process) error {
	if c.State.IsRunning() {
		return nil
	}

	/*
		configPath, err := d.generateConfig(c)
		if err != nil {
			return err
		}
	*/

	params := []string{
		startPath,
		"-n", c.Name,
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
		params = append(params, "-privileged")
	}

	if c.WorkingDir != "" {
		params = append(params, "-w", c.WorkingDir)
	}

	params = append(params, "--", c.Entrypoint)
	params = append(params, c.Args...)

	cmd := exec.Command(params[0], params[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := c.SetCmd(cmd); err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Poll for running
	if err := d.waitForStart(cmd, c); err != nil {
		return err
	}
	return nil
}

func (d *driver) Stop(c *execdriver.Process) error {
	if err := d.kill(c, 15); err != nil {
		if err := d.kill(c, 9); err != nil {
			return err
		}
	}

	if err := d.wait(c, 10*time.Second); err != nil {
		if err := d.kill(c, 9); err != nil {
			return err
		}
	}
	exitCode := c.GetExitCode()
	if err := c.State.SetStopped(exitCode); err != nil {
		return err
	}
	return nil
}

func (d *driver) Kill(c *execdriver.Process, sig int) error {
	c.State.Lock()
	defer c.State.Unlock()
	return d.kill(c, sig)
}

func (d *driver) Wait(c *execdriver.Process, duration time.Duration) error {
	return d.wait(c, duration)
}

// If seconds < 0 then wait forever
func (d *driver) wait(c *execdriver.Process, duration time.Duration) error {
begin:
	var (
		killer bool
		done   = d.waitCmd(c)
	)
	if duration > 0 {
		select {
		case err := <-done:
			if err != nil && err == execdriver.ErrCommandIsNil {
				done = d.waitLxc(c, &killer)
				goto begin
			}
			return err
		case <-time.After(duration):
			killer = true
			return ErrWaitTimeoutReached
		}
	} else {
		if err := <-done; err != nil {
			if err == execdriver.ErrCommandIsNil {
				done = d.waitLxc(c, &killer)
				goto begin
			}
			return err
		}
	}
	return nil
}

func (d *driver) kill(c *execdriver.Process, sig int) error {
	return exec.Command("lxc-kill", "-n", c.Name, strconv.Itoa(sig)).Run()
}

/* Generate the lxc configuration and return the path to the file
func (d *driver) generateConfig(c *execdriver.Process) (string, error) {
	p := path.Join(d.root, c.Name)
	f, err := os.Create(p)
	if err != nil {
		return "", nil
	}
	defer f.Close()

	if err := LxcTemplateCompiled.Execute(f, c.Context); err != nil {
		return "", err
	}
	return p, nil
}
*/

func (d *driver) waitForStart(cmd *exec.Cmd, c *execdriver.Process) error {
	// We wait for the container to be fully running.
	// Timeout after 5 seconds. In case of broken pipe, just retry.
	// Note: The container can run and finish correctly before
	// the end of this loop
	for now := time.Now(); time.Since(now) < 5*time.Second; {
		// If the container dies while waiting for it, just return
		if !c.State.IsRunning() {
			return nil
		}
		output, err := exec.Command("lxc-info", "-s", "-n", c.Name).CombinedOutput()
		if err != nil {
			output, err = exec.Command("lxc-info", "-s", "-n", c.Name).CombinedOutput()
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

func (d *driver) waitCmd(c *execdriver.Process) <-chan error {
	done := make(chan error)
	go func() {
		done <- c.Wait()
	}()
	return done
}

func (d *driver) waitLxc(c *execdriver.Process, kill *bool) <-chan error {
	done := make(chan error)
	go func() {
		for *kill {
			output, err := exec.Command("lxc-info", "-n", c.Name).CombinedOutput()
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
