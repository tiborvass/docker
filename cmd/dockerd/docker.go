package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/dockerversion"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/pkg/reexec"
	"github.com/tiborvass/docker/pkg/term"
	"github.com/tiborvass/docker/utils"
)

var (
	daemonCli = NewDaemonCli()
	flHelp    = flag.Bool([]string{"h", "-help"}, false, "Print usage")
	flVersion = flag.Bool([]string{"v", "-version"}, false, "Print version information and quit")
)

func main() {
	if reexec.Init() {
		return
	}

	// Set terminal emulation based on platform as required.
	_, stdout, stderr := term.StdStreams()

	logrus.SetOutput(stderr)

	flag.Merge(flag.CommandLine, daemonCli.commonFlags.FlagSet)

	flag.Usage = func() {
		fmt.Fprint(stdout, "Usage: dockerd [ --help | -v | --version ]\n\n")
		fmt.Fprint(stdout, "A self-sufficient runtime for containers.\n\nOptions:\n")

		flag.CommandLine.SetOutput(stdout)
		flag.PrintDefaults()
	}
	flag.CommandLine.ShortUsage = func() {
		fmt.Fprint(stderr, "\nUsage:\tdockerd [OPTIONS]\n")
	}

	if err := flag.CommandLine.ParseFlags(os.Args[1:], false); err != nil {
		os.Exit(1)
	}

	if *flVersion {
		showVersion()
		return
	}

	if *flHelp {
		// if global flag --help is present, regardless of what other options and commands there are,
		// just print the usage.
		flag.Usage()
		return
	}
	daemonCli.start()
}

func showVersion() {
	if utils.ExperimentalBuild() {
		fmt.Printf("Docker version %s, build %s, experimental\n", dockerversion.Version, dockerversion.GitCommit)
	} else {
		fmt.Printf("Docker version %s, build %s\n", dockerversion.Version, dockerversion.GitCommit)
	}
}
