package client

import (
	"fmt"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/tiborvass/docker/engine"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/utils"
)

func (cli *DockerCli) CmdTop(args ...string) error {
	cmd := cli.Subcmd("top", "CONTAINER [ps OPTIONS]", "Display the running processes of a container", true)
	cmd.Require(flag.Min, 1)

	utils.ParseFlags(cmd, args, true)

	val := url.Values{}
	if cmd.NArg() > 1 {
		val.Set("ps_args", strings.Join(cmd.Args()[1:], " "))
	}

	stream, _, err := cli.call("GET", "/containers/"+cmd.Arg(0)+"/top?"+val.Encode(), nil, false)
	if err != nil {
		return err
	}
	var procs engine.Env
	if err := procs.Decode(stream); err != nil {
		return err
	}
	w := tabwriter.NewWriter(cli.out, 20, 1, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(procs.GetList("Titles"), "\t"))
	processes := [][]string{}
	if err := procs.GetJson("Processes", &processes); err != nil {
		return err
	}
	for _, proc := range processes {
		fmt.Fprintln(w, strings.Join(proc, "\t"))
	}
	w.Flush()
	return nil
}
