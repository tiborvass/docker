package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestDaemonRestartWithRunningContainersPorts(t *testing.T) {
	d := NewDaemon(t)
	if err := d.StartWithBusybox(); err != nil {
		t.Fatalf("Could not start daemon with busybox: %v", err)
	}
	defer d.Stop()

	if out, err := d.Cmd("run", "-d", "--name", "top1", "-p", "1234:80", "--restart", "always", "busybox:latest", "top"); err != nil {
		t.Fatalf("Could not run top1: err=%v\n%s", err, out)
	}
	// --restart=no by default
	if out, err := d.Cmd("run", "-d", "--name", "top2", "-p", "80", "busybox:latest", "top"); err != nil {
		t.Fatalf("Could not run top2: err=%v\n%s", err, out)
	}

	testRun := func(m map[string]bool, prefix string) {
		var format string
		for c, shouldRun := range m {
			out, err := d.Cmd("ps")
			if err != nil {
				t.Fatalf("Could not run ps: err=%v\n%q", err, out)
			}
			if shouldRun {
				format = "%scontainer %q is not running"
			} else {
				format = "%scontainer %q is running"
			}
			if shouldRun != strings.Contains(out, c) {
				t.Fatalf(format, prefix, c)
			}
		}
	}

	testRun(map[string]bool{"top1": true, "top2": true}, "")

	if err := d.Restart(); err != nil {
		t.Fatalf("Could not restart daemon: %v", err)
	}

	testRun(map[string]bool{"top1": true, "top2": false}, "After daemon restart: ")

	logDone("daemon - running containers on daemon restart")
}

func TestDaemonRestartWithVolumesRefs(t *testing.T) {
	d := NewDaemon(t)
	if err := d.StartWithBusybox(); err != nil {
		t.Fatal(err)
	}
	defer d.Stop()

	if out, err := d.Cmd("run", "-d", "--name", "volrestarttest1", "-v", "/foo", "busybox"); err != nil {
		t.Fatal(err, out)
	}
	if err := d.Restart(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Cmd("run", "-d", "--volumes-from", "volrestarttest1", "--name", "volrestarttest2", "busybox"); err != nil {
		t.Fatal(err)
	}
	if out, err := d.Cmd("rm", "-fv", "volrestarttest2"); err != nil {
		t.Fatal(err, out)
	}
	v, err := d.Cmd("inspect", "--format", "{{ json .Volumes }}", "volrestarttest1")
	if err != nil {
		t.Fatal(err)
	}
	volumes := make(map[string]string)
	json.Unmarshal([]byte(v), &volumes)
	if _, err := os.Stat(volumes["/foo"]); err != nil {
		t.Fatalf("Expected volume to exist: %s - %s", volumes["/foo"], err)
	}

	logDone("daemon - volume refs are restored")
}

func TestDaemonStartIptablesFalse(t *testing.T) {
	d := NewDaemon(t)
	if err := d.Start("--iptables=false"); err != nil {
		t.Fatalf("we should have been able to start the daemon with passing iptables=false: %v", err)
	}
	d.Stop()

	logDone("daemon - started daemon with iptables=false")
}
