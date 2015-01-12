package client

import (
	"fmt"

	"github.com/tiborvass/docker/engine"
	"github.com/tiborvass/docker/pkg/archive"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/utils"
)

// CmdDiff shows changes on a container's filesystem.
//
// Each changed file is printed on a separate line, prefixed with a single character that indicates the status of the file: C (modified), A (added), or D (deleted).
//
// Usage: docker diff CONTAINER
func (cli *DockerCli) CmdDiff(args ...string) error {
	cmd := cli.Subcmd("diff", "CONTAINER", "Inspect changes on a container's filesystem", true)
	cmd.Require(flag.Exact, 1)

	utils.ParseFlags(cmd, args, true)

	body, _, err := readBody(cli.call("GET", "/containers/"+cmd.Arg(0)+"/changes", nil, nil))

	if err != nil {
		return err
	}

	outs := engine.NewTable("", 0)
	if _, err := outs.ReadListFrom(body); err != nil {
		return err
	}
	for _, change := range outs.Data {
		var kind string
		switch change.GetInt("Kind") {
		case archive.ChangeModify:
			kind = "C"
		case archive.ChangeAdd:
			kind = "A"
		case archive.ChangeDelete:
			kind = "D"
		}
		fmt.Fprintf(cli.out, "%s %s\n", kind, change.Get("Path"))
	}
	return nil
}
