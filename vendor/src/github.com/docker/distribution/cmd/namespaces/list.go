package main

import (
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/docker/distribution/namespace"
)

var (
	commandList = cli.Command{
		Name:   "list",
		Usage:  `List the current namespace configuration, optionally filtered by arguments.`,
		Action: list,
	}
)

func list(ctx *cli.Context) {
	entries, err := namespaces.Find(ctx.Args()...)
	if err != nil {
		errorf("error finding %v: %v", ctx.Args(), err)
	}

	// if err := WriteManager(os.Stdout, &entries); err != nil {
	// 	log.Fatalln(err)
	// }

	// sort.Stable(entries)

	if err := namespace.WriteEntries(os.Stdout, entries); err != nil {
		log.Fatalln(err)
	}
}
