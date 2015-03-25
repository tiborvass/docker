package client

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/tiborvass/docker/engine"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/tiborvass/docker/pkg/units"
	"github.com/tiborvass/docker/utils"
)

// CmdHistory shows the history of an image.
//
// Usage: docker history [OPTIONS] IMAGE
func (cli *DockerCli) CmdHistory(args ...string) error {
	cmd := cli.Subcmd("history", "IMAGE", "Show the history of an image", true)
	quiet := cmd.Bool([]string{"q", "-quiet"}, false, "Only show numeric IDs")
	noTrunc := cmd.Bool([]string{"#notrunc", "-no-trunc"}, false, "Don't truncate output")
	cmd.Require(flag.Exact, 1)

	utils.ParseFlags(cmd, args, true)

	body, _, err := readBody(cli.call("GET", "/images/"+cmd.Arg(0)+"/history", nil, false))
	if err != nil {
		return err
	}

	outs := engine.NewTable("Created", 0)
	if _, err := outs.ReadListFrom(body); err != nil {
		return err
	}

	w := tabwriter.NewWriter(cli.out, 20, 1, 3, ' ', 0)
	if !*quiet {
		fmt.Fprintln(w, "IMAGE\tCREATED\tCREATED BY\tSIZE")
	}

	for _, out := range outs.Data {
		outID := out.Get("Id")
		if !*quiet {
			if *noTrunc {
				fmt.Fprintf(w, "%s\t", outID)
			} else {
				fmt.Fprintf(w, "%s\t", stringid.TruncateID(outID))
			}

			fmt.Fprintf(w, "%s ago\t", units.HumanDuration(time.Now().UTC().Sub(time.Unix(out.GetInt64("Created"), 0))))

			if *noTrunc {
				fmt.Fprintf(w, "%s\t", out.Get("CreatedBy"))
			} else {
				fmt.Fprintf(w, "%s\t", utils.Trunc(out.Get("CreatedBy"), 45))
			}
			fmt.Fprintf(w, "%s\n", units.HumanSize(float64(out.GetInt64("Size"))))
		} else {
			if *noTrunc {
				fmt.Fprintln(w, outID)
			} else {
				fmt.Fprintln(w, stringid.TruncateID(outID))
			}
		}
	}
	w.Flush()
	return nil
}
