package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/types"
	Cli "github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/opts"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/pkg/parsers/filters"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/tiborvass/docker/pkg/units"
)

// CmdImages lists the images in a specified repository, or all top-level images if no repository is specified.
//
// Usage: docker images [OPTIONS] [REPOSITORY]
func (cli *DockerCli) CmdImages(args ...string) error {
	cmd := Cli.Subcmd("images", []string{"[REPOSITORY[:TAG]]"}, Cli.DockerCommands["images"].Description, true)
	quiet := cmd.Bool([]string{"q", "-quiet"}, false, "Only show numeric IDs")
	all := cmd.Bool([]string{"a", "-all"}, false, "Show all images (default hides intermediate images)")
	noTrunc := cmd.Bool([]string{"-no-trunc"}, false, "Don't truncate output")
	showDigests := cmd.Bool([]string{"-digests"}, false, "Show digests")

	flFilter := opts.NewListOpts(nil)
	cmd.Var(&flFilter, []string{"f", "-filter"}, "Filter output based on conditions provided")
	cmd.Require(flag.Max, 1)

	cmd.ParseFlags(args, true)

	// Consolidate all filter flags, and sanity check them early.
	// They'll get process in the daemon/server.
	imageFilterArgs := filters.Args{}
	for _, f := range flFilter.GetAll() {
		var err error
		imageFilterArgs, err = filters.ParseFlag(f, imageFilterArgs)
		if err != nil {
			return err
		}
	}

	matchName := cmd.Arg(0)
	v := url.Values{}
	if len(imageFilterArgs) > 0 {
		filterJSON, err := filters.ToParam(imageFilterArgs)
		if err != nil {
			return err
		}
		v.Set("filters", filterJSON)
	}

	if cmd.NArg() == 1 {
		// FIXME rename this parameter, to not be confused with the filters flag
		v.Set("filter", matchName)
	}
	if *all {
		v.Set("all", "1")
	}

	serverResp, err := cli.call("GET", "/images/json?"+v.Encode(), nil, nil)
	if err != nil {
		return err
	}

	defer serverResp.body.Close()

	images := []types.Image{}
	if err := json.NewDecoder(serverResp.body).Decode(&images); err != nil {
		return err
	}

	w := tabwriter.NewWriter(cli.out, 20, 1, 3, ' ', 0)
	if !*quiet {
		if *showDigests {
			fmt.Fprintln(w, "REPOSITORY\tTAG\tDIGEST\tIMAGE ID\tCREATED\tSIZE")
		} else {
			fmt.Fprintln(w, "REPOSITORY\tTAG\tIMAGE ID\tCREATED\tSIZE")
		}
	}

	for _, image := range images {
		ID := image.ID
		if !*noTrunc {
			ID = stringid.TruncateID(ID)
		}

		repoTags := image.RepoTags
		repoDigests := image.RepoDigests

		if len(repoTags) == 1 && repoTags[0] == "<none>:<none>" && len(repoDigests) == 1 && repoDigests[0] == "<none>@<none>" {
			// dangling image - clear out either repoTags or repoDigsts so we only show it once below
			repoDigests = []string{}
		}

		// combine the tags and digests lists
		tagsAndDigests := append(repoTags, repoDigests...)
		for _, repoAndRef := range tagsAndDigests {
			// default repo, tag, and digest to none - if there's a value, it'll be set below
			repo := "<none>"
			tag := "<none>"
			digest := "<none>"

			if !strings.HasPrefix(repoAndRef, "<none>") {
				ref, err := reference.ParseNamed(repoAndRef)
				if err != nil {
					return err
				}
				repo = ref.Name()

				switch x := ref.(type) {
				case reference.Digested:
					digest = x.Digest().String()
				case reference.Tagged:
					tag = x.Tag()
				}
			}

			if !*quiet {
				if *showDigests {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s ago\t%s\n", repo, tag, digest, ID, units.HumanDuration(time.Now().UTC().Sub(time.Unix(int64(image.Created), 0))), units.HumanSize(float64(image.Size)))
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s ago\t%s\n", repo, tag, ID, units.HumanDuration(time.Now().UTC().Sub(time.Unix(int64(image.Created), 0))), units.HumanSize(float64(image.Size)))
				}
			} else {
				fmt.Fprintln(w, ID)
			}
		}
	}

	if !*quiet {
		w.Flush()
	}
	return nil
}
