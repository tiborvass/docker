package client

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/tiborvass/docker/api/types"
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
	cmd.ParseFlags(args, true)

	rdr, _, err := cli.call("GET", "/images/"+cmd.Arg(0)+"/history", nil, nil)
	if err != nil {
		return err
	}

	history := []types.ImageHistory{}
	err = json.NewDecoder(rdr).Decode(&history)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cli.out, 20, 1, 3, ' ', 0)
	if !*quiet {
		fmt.Fprintln(w, "IMAGE\tCREATED\tCREATED BY\tSIZE\tCOMMENT")
	}

	for _, entry := range history {
		if *noTrunc {
			fmt.Fprintf(w, entry.ID)
		} else {
			fmt.Fprintf(w, stringid.TruncateID(entry.ID))
		}
		if !*quiet {
			fmt.Fprintf(w, "\t%s ago\t", units.HumanDuration(time.Now().UTC().Sub(time.Unix(entry.Created, 0))))

			if *noTrunc {
				fmt.Fprintf(w, "%s\t", entry.CreatedBy)
			} else {
				fmt.Fprintf(w, "%s\t", utils.Trunc(entry.CreatedBy, 45))
			}
			fmt.Fprintf(w, "%s\t", units.HumanSize(float64(entry.Size)))
			fmt.Fprintf(w, "%s\n", entry.Comment)
		}
		fmt.Fprintf(w, "\n")
	}
	w.Flush()
	return nil
}
