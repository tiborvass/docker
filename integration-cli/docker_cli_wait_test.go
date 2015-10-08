package main

import (
	"bytes"
	"os/exec"
	"strings"
	"time"

	"github.com/go-check/check"
)

// non-blocking wait with 0 exit code
func (s *DockerSuite) TestWaitNonBlockedExitZero(c *check.C) {
	out, _ := dockerCmd(c, "run", "-d", "busybox", "sh", "-c", "true")
	containerID := strings.TrimSpace(out)

	if err := waitInspect(containerID, "{{.State.Running}}", "false", 30*time.Second); err != nil {
		c.Fatal("Container should have stopped by now")
	}

	out, _ = dockerCmd(c, "wait", containerID)
	if strings.TrimSpace(out) != "0" {
		c.Fatal("failed to set up container", out)
	}

}

// blocking wait with 0 exit code
func (s *DockerSuite) TestWaitBlockedExitZero(c *check.C) {
	// Windows busybox does not support trap in this way, not sleep with sub-second
	// granularity. It will always exit 0x40010004.
	testRequires(c, DaemonIsLinux)
	out, _ := dockerCmd(c, "run", "-d", "busybox", "/bin/sh", "-c", "trap 'exit 0' TERM; while true; do usleep 10; done")
	containerID := strings.TrimSpace(out)

	c.Assert(waitRun(containerID), check.IsNil)

	chWait := make(chan string)
	go func() {
		out, _, _ := runCommandWithOutput(exec.Command(dockerBinary, "wait", containerID))
		chWait <- out
	}()

	time.Sleep(100 * time.Millisecond)
	dockerCmd(c, "stop", containerID)

	select {
	case status := <-chWait:
		if strings.TrimSpace(status) != "0" {
			c.Fatalf("expected exit 0, got %s", status)
		}
	case <-time.After(2 * time.Second):
		c.Fatal("timeout waiting for `docker wait` to exit")
	}

}

// non-blocking wait with random exit code
func (s *DockerSuite) TestWaitNonBlockedExitRandom(c *check.C) {
	out, _ := dockerCmd(c, "run", "-d", "busybox", "sh", "-c", "exit 99")
	containerID := strings.TrimSpace(out)

	if err := waitInspect(containerID, "{{.State.Running}}", "false", 30*time.Second); err != nil {
		c.Fatal("Container should have stopped by now")
	}

	out, _ = dockerCmd(c, "wait", containerID)
	if strings.TrimSpace(out) != "99" {
		c.Fatal("failed to set up container", out)
	}

}

// blocking wait with random exit code
func (s *DockerSuite) TestWaitBlockedExitRandom(c *check.C) {
	// Cannot run on Windows as trap in Windows busybox does not support trap in this way.
	testRequires(c, DaemonIsLinux)
	out, _ := dockerCmd(c, "run", "-d", "busybox", "/bin/sh", "-c", "trap 'exit 99' TERM; while true; do usleep 10; done")
	containerID := strings.TrimSpace(out)
	c.Assert(waitRun(containerID), check.IsNil)

	chWait := make(chan error)
	waitCmd := exec.Command(dockerBinary, "wait", containerID)
	waitCmdOut := bytes.NewBuffer(nil)
	waitCmd.Stdout = waitCmdOut
	if err := waitCmd.Start(); err != nil {
		c.Fatal(err)
	}

	go func() {
		chWait <- waitCmd.Wait()
	}()

	dockerCmd(c, "stop", containerID)

	select {
	case err := <-chWait:
		if err != nil {
			c.Fatal(err)
		}
		status, err := waitCmdOut.ReadString('\n')
		if err != nil {
			c.Fatal(err)
		}
		if strings.TrimSpace(status) != "99" {
			c.Fatalf("expected exit 99, got %s", status)
		}
	case <-time.After(2 * time.Second):
		waitCmd.Process.Kill()
		c.Fatal("timeout waiting for `docker wait` to exit")
	}
}
