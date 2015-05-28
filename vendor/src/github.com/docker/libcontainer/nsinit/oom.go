package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var oomCommand = cli.Command{
	Name:  "oom",
	Usage: "display oom notifications for a container",
	Flags: []cli.Flag{
		cli.StringFlag{Name: "id", Value: "nsinit", Usage: "specify the ID for a container"},
	},
	Action: func(context *cli.Context) {
		container, err := getContainer(context)
		if err != nil {
			logrus.Fatal(err)
		}
		n, err := container.NotifyOOM()
		if err != nil {
			logrus.Fatal(err)
		}
		for x := range n {
			// hack for calm down go1.4 gofmt
			_ = x
			logrus.Printf("OOM notification received")
		}
	},
}
