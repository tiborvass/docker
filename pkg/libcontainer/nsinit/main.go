package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"

	"github.com/dotcloud/docker/pkg/libcontainer"
	"github.com/dotcloud/docker/pkg/libcontainer/cgroups/fs"
	"github.com/dotcloud/docker/pkg/libcontainer/namespaces"
)

var (
	dataPath  = os.Getenv("data_path")
	console   = os.Getenv("console")
	rawPipeFd = os.Getenv("pipe")
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("invalid number of arguments %d", len(os.Args))
	}

	switch os.Args[1] {
	case "exec": // this is executed outside of the namespace in the cwd
		container, err := loadContainer()
		if err != nil {
			log.Fatalf("unable to load container: %s", err)
		}

		var nspid, exitCode int
		if nspid, err = readPid(); err != nil && !os.IsNotExist(err) {
			log.Fatalf("unable to read pid: %s", err)
		}

		if nspid > 0 {
			err = namespaces.ExecIn(container, nspid, os.Args[2:])
		} else {
			term := namespaces.NewTerminal(os.Stdin, os.Stdout, os.Stderr, container.Tty)
			exitCode, err = startContainer(container, term, dataPath, os.Args[2:])
		}

		if err != nil {
			log.Fatalf("failed to exec: %s", err)
		}
		os.Exit(exitCode)
	case "nsenter": // this is executed inside the namespace.
		// nsinit nsenter <pid> <process label> <container JSON> <cmd>...
		if len(os.Args) < 6 {
			log.Fatalf("incorrect usage: nsinit nsenter <pid> <process label> <container JSON> <cmd>...")
		}

		container, err := loadContainerFromJson(os.Args[4])
		if err != nil {
			log.Fatalf("unable to load container: %s", err)
		}

		nspid, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("unable to read pid: %s from %q", err, os.Args[2])
		}

		if nspid <= 0 {
			log.Fatalf("cannot enter into namespaces without valid pid: %q", nspid)
		}

		err = namespaces.NsEnter(container, os.Args[3], nspid, os.Args[5:])
		if err != nil {
			log.Fatalf("failed to nsenter: %s", err)
		}
	case "init": // this is executed inside of the namespace to setup the container
		container, err := loadContainer()
		if err != nil {
			log.Fatalf("unable to load container: %s", err)
		}

		// by default our current dir is always our rootfs
		rootfs, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		pipeFd, err := strconv.Atoi(rawPipeFd)
		if err != nil {
			log.Fatal(err)
		}
		syncPipe, err := namespaces.NewSyncPipeFromFd(0, uintptr(pipeFd))
		if err != nil {
			log.Fatalf("unable to create sync pipe: %s", err)
		}

		if err := namespaces.Init(container, rootfs, console, syncPipe, os.Args[2:]); err != nil {
			log.Fatalf("unable to initialize for container: %s", err)
		}
	case "stats":
		container, err := loadContainer()
		if err != nil {
			log.Fatalf("unable to load container: %s", err)
		}

		// returns the stats of the current container.
		stats, err := getContainerStats(container)
		if err != nil {
			log.Printf("Failed to get stats - %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Stats:\n%v\n", stats)
		os.Exit(0)

	case "spec":
		container, err := loadContainer()
		if err != nil {
			log.Fatalf("unable to load container: %s", err)
		}

		// returns the spec of the current container.
		spec, err := getContainerSpec(container)
		if err != nil {
			log.Printf("Failed to get spec - %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Spec:\n%v\n", spec)
		os.Exit(0)

	default:
		log.Fatalf("command not supported for nsinit %s", os.Args[1])
	}
}

func loadContainer() (*libcontainer.Container, error) {
	f, err := os.Open(filepath.Join(dataPath, "container.json"))
	if err != nil {
		log.Printf("Path: %q", filepath.Join(dataPath, "container.json"))
		return nil, err
	}
	defer f.Close()

	var container *libcontainer.Container
	if err := json.NewDecoder(f).Decode(&container); err != nil {
		return nil, err
	}
	return container, nil
}

func loadContainerFromJson(rawData string) (*libcontainer.Container, error) {
	container := &libcontainer.Container{}
	err := json.Unmarshal([]byte(rawData), container)
	if err != nil {
		return nil, err
	}
	return container, nil
}

func readPid() (int, error) {
	data, err := ioutil.ReadFile(filepath.Join(dataPath, "pid"))
	if err != nil {
		return -1, err
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return -1, err
	}
	return pid, nil
}

// startContainer starts the container. Returns the exit status or -1 and an
// error.
//
// Signals sent to the current process will be forwarded to container.
func startContainer(container *libcontainer.Container, term namespaces.Terminal, dataPath string, args []string) (int, error) {
	var (
		cmd  *exec.Cmd
		sigc = make(chan os.Signal, 10)
	)

	signal.Notify(sigc)

	createCommand := func(container *libcontainer.Container, console, rootfs, dataPath, init string, pipe *os.File, args []string) *exec.Cmd {
		cmd = namespaces.DefaultCreateCommand(container, console, rootfs, dataPath, init, pipe, args)
		return cmd
	}

	startCallback := func() {
		go func() {
			for sig := range sigc {
				cmd.Process.Signal(sig)
			}
		}()
	}

	return namespaces.Exec(container, term, "", dataPath, args, createCommand, startCallback)
}

// returns the container stats in json format.
func getContainerStats(container *libcontainer.Container) (string, error) {
	stats, err := fs.GetStats(container.Cgroups)
	if err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(stats, "", "\t")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// returns the container spec in json format.
func getContainerSpec(container *libcontainer.Container) (string, error) {
	spec, err := json.MarshalIndent(container, "", "\t")
	if err != nil {
		return "", err
	}
	return string(spec), nil
}
