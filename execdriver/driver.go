package execdriver

import (
	"errors"
	"io"
	"net"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var (
	ErrCommandIsNil = errors.New("Process's cmd is nil")
)

type Driver interface {
	Start(c *Process) error
	Stop(c *Process) error
	Kill(c *Process, sig int) error
	Wait(c *Process, duration time.Duration) error
}

// Network settings of the container
type Network struct {
	Gateway     string
	IPAddress   net.IPAddr
	IPPrefixLen int
	Mtu         int
}

type State struct {
	sync.RWMutex
	running    bool
	pid        int
	exitCode   int
	startedAt  time.Time
	finishedAt time.Time
}

func (s *State) IsRunning() bool {
	s.RLock()
	defer s.RUnlock()
	return s.running
}

func (s *State) SetRunning() error {
	s.Lock()
	defer s.Unlock()
	s.running = true
	return nil
}

func (s *State) SetStopped(exitCode int) error {
	s.Lock()
	defer s.Unlock()
	s.exitCode = exitCode
	s.running = false
	return nil
}

// Container / Process / Whatever, we can redefine the conatiner here
// to be what it should be and not have to carry the baggage of the
// container type in the core with backward compat.  This is what a
// driver needs to execute a process inside of a conatiner.  This is what
// a container is at it's core.
type Process struct {
	State State

	Name        string // unique name for the conatienr
	Privileged  bool
	User        string
	Dir         string // root fs of the container
	InitPath    string // dockerinit
	Entrypoint  string
	Args        []string
	Environment map[string]string
	WorkingDir  string
	ConfigPath  string
	Network     *Network // if network is nil then networking is disabled
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer

	cmd *exec.Cmd
}

func (c *Process) SetCmd(cmd *exec.Cmd) error {
	c.cmd = cmd
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	cmd.Stdin = c.Stdin
	return nil
}

func (c *Process) StdinPipe() (io.WriteCloser, error) {
	return c.cmd.StdinPipe()
}

func (c *Process) StderrPipe() (io.ReadCloser, error) {
	return c.cmd.StderrPipe()
}

func (c *Process) StdoutPipe() (io.ReadCloser, error) {
	return c.cmd.StdoutPipe()
}

func (c *Process) GetExitCode() int {
	if c.cmd != nil {
		return c.cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
	}
	return -1
}

func (c *Process) Wait() error {
	if c.cmd != nil {
		return c.cmd.Wait()
	}
	return ErrCommandIsNil
}
