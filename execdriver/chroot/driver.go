package chroot

import (
	"fmt"
	"github.com/dotcloud/docker/execdriver"
	"github.com/dotcloud/docker/pkg/mount"
	"os"
	"os/exec"
)

const (
	DriverName = "chroot"
	Version    = "0.1"
)

func init() {
	execdriver.RegisterInitFunc(DriverName, func(args *execdriver.InitArgs) error {
		if err := mount.ForceMount("proc", "proc", "proc", ""); err != nil {
			return err
		}
		defer mount.ForceUnmount("proc")
		cmd := exec.Command(args.Args[0], args.Args[1:]...)

		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin

		return cmd.Run()
	})
}

type driver struct {
}

func NewDriver() (*driver, error) {
	return &driver{}, nil
}

func (d *driver) Run(c *execdriver.Process, startCallback execdriver.StartCallback) (int, error) {
	params := []string{
		"chroot",
		c.Rootfs,
		"/.dockerinit",
		"-driver",
		DriverName,
	}
	params = append(params, c.Entrypoint)
	params = append(params, c.Arguments...)

	var (
		name = params[0]
		arg  = params[1:]
	)
	aname, err := exec.LookPath(name)
	if err != nil {
		aname = name
	}
	c.Path = aname
	c.Args = append([]string{name}, arg...)

	if err := c.Start(); err != nil {
		return -1, err
	}

	if startCallback != nil {
		startCallback(c)
	}

	err = c.Wait()
	return c.GetExitCode(), err
}

func (d *driver) Kill(p *execdriver.Process, sig int) error {
	return p.Process.Kill()
}

func (d *driver) Wait(id string) error {
	panic("Not Implemented")
}

func (d *driver) Info(id string) execdriver.Info {
	panic("Not implemented")
}

func (d *driver) Name() string {
	return fmt.Sprintf("%s-%s", DriverName, Version)
}
