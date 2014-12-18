// +build linux

package native

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/docker/libcontainer"
)

func findUserArgs() []string {
	for i, a := range os.Args {
		if a == "--" {
			return os.Args[i+1:]
		}
	}
	return []string{}
}

// loadConfigFromFd loads a container's config from the sync pipe that is provided by
// fd 3 when running a process
func loadConfigFromFd() (*libcontainer.Config, error) {
	var config *libcontainer.Config
	if err := json.NewDecoder(os.NewFile(3, "child")).Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}

var nsFlags = map[string]int{
	"net":  syscall.CLONE_NEWNET,
	"ipc":  syscall.CLONE_NEWIPC,
	"ns":   syscall.CLONE_NEWNS,
	"pid":  syscall.CLONE_NEWPID,
	"user": syscall.CLONE_NEWUSER,
	"uts":  syscall.CLONE_NEWUTS,
}

func init() { reexec.Register("docker-createns", createNamespace) }
func createNamespace() {
	runtime.LockOSThread()

	path := flag.String("path", "", "path to bind mount file")
	ns := flag.String("ns", "", "Namespace type")

	flag.Parse()

	flags, ok := nsFlags[*ns]
	if !ok {
		log.Fatalf("Unknown namespace: %s", ns)
	}

	if err := syscall.Unshare(flags); err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(*path)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	if os.IsExist(err) {
		os.Exit(0)
	}
	f.Close()

	if err := syscall.Mount(fmt.Sprintf("/proc/self/ns/%s", *ns), *path, "bind", syscall.MS_BIND, ""); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

func CreateNamespace(ns, path string) error {
	cmd := &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{
			"docker-createns",
			"-ns", ns,
			"-path", path,
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd.Run()
}
