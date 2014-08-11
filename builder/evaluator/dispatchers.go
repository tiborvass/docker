package evaluator

// This file contains the dispatchers for each command. Note that
// `nullDispatch` is not actually a command, but support for commands we parse
// but do nothing with.
//
// See evaluator.go for a higher level discussion of the whole evaluator
// package.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tiborvass/docker/nat"
	"github.com/tiborvass/docker/runconfig"
	"github.com/tiborvass/docker/utils"
)

// dispatch with no layer / parsing. This is effectively not a command.
func nullDispatch(b *BuildFile, args []string) error {
	return nil
}

// ENV foo bar
//
// Sets the environment variable foo to bar, also makes interpolation
// in the dockerfile available from the next statement on via ${foo}.
//
func env(b *BuildFile, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("ENV accepts two arguments")
	}

	// the duplication here is intended to ease the replaceEnv() call's env
	// handling. This routine gets much shorter with the denormalization here.
	key := args[0]
	b.Env[key] = args[1]
	b.Config.Env = append(b.Config.Env, strings.Join([]string{key, b.Env[key]}, "="))

	return b.commit("", b.Config.Cmd, fmt.Sprintf("ENV %s=%s", key, b.Env[key]))
}

// MAINTAINER some text <maybe@an.email.address>
//
// Sets the maintainer metadata.
func maintainer(b *BuildFile, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("MAINTAINER requires only one argument")
	}

	b.maintainer = args[0]
	return b.commit("", b.Config.Cmd, fmt.Sprintf("MAINTAINER %s", b.maintainer))
}

// ADD foo /path
//
// Add the file 'foo' to '/path'. Tarball and Remote URL (git, http) handling
// exist here. If you do not wish to have this automatic handling, use COPY.
//
func add(b *BuildFile, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("ADD requires two arguments")
	}

	return b.runContextCommand(args, true, true, "ADD")
}

// COPY foo /path
//
// Same as 'ADD' but without the tar and remote url handling.
//
func dispatchCopy(b *BuildFile, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("COPY requires two arguments")
	}

	return b.runContextCommand(args, false, false, "COPY")
}

// FROM imagename
//
// This sets the image the dockerfile will build on top of.
//
func from(b *BuildFile, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("FROM requires one argument")
	}

	name := args[0]

	image, err := b.Options.Daemon.Repositories().LookupImage(name)
	if err != nil {
		if b.Options.Daemon.Graph().IsNotExist(err) {
			image, err = b.pullImage(name)
		}

		// note that the top level err will still be !nil here if IsNotExist is
		// not the error. This approach just simplifies hte logic a bit.
		if err != nil {
			return err
		}
	}

	return b.processImageFrom(image)
}

// ONBUILD RUN echo yo
//
// ONBUILD triggers run when the image is used in a FROM statement.
//
// ONBUILD handling has a lot of special-case functionality, the heading in
// evaluator.go and comments around dispatch() in the same file explain the
// special cases. search for 'OnBuild' in internals.go for additional special
// cases.
//
func onbuild(b *BuildFile, args []string) error {
	triggerInstruction := strings.ToUpper(strings.TrimSpace(args[0]))
	switch triggerInstruction {
	case "ONBUILD":
		return fmt.Errorf("Chaining ONBUILD via `ONBUILD ONBUILD` isn't allowed")
	case "MAINTAINER", "FROM":
		return fmt.Errorf("%s isn't allowed as an ONBUILD trigger", triggerInstruction)
	}

	trigger := strings.Join(args, " ")

	b.Config.OnBuild = append(b.Config.OnBuild, trigger)
	return b.commit("", b.Config.Cmd, fmt.Sprintf("ONBUILD %s", trigger))
}

// WORKDIR /tmp
//
// Set the working directory for future RUN/CMD/etc statements.
//
func workdir(b *BuildFile, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("WORKDIR requires exactly one argument")
	}

	workdir := args[0]

	if workdir[0] == '/' {
		b.Config.WorkingDir = workdir
	} else {
		if b.Config.WorkingDir == "" {
			b.Config.WorkingDir = "/"
		}
		b.Config.WorkingDir = filepath.Join(b.Config.WorkingDir, workdir)
	}

	return b.commit("", b.Config.Cmd, fmt.Sprintf("WORKDIR %v", workdir))
}

// RUN some command yo
//
// run a command and commit the image. Args are automatically prepended with
// 'sh -c' in the event there is only one argument. The difference in
// processing:
//
// RUN echo hi          # sh -c echo hi
// RUN [ "echo", "hi" ] # echo hi
//
func run(b *BuildFile, args []string) error {
	if len(args) == 1 { // literal string command, not an exec array
		args = append([]string{"/bin/sh", "-c"}, args[0])
	}

	if b.image == "" {
		return fmt.Errorf("Please provide a source image with `from` prior to run")
	}

	config, _, _, err := runconfig.Parse(append([]string{b.image}, args...), nil)
	if err != nil {
		return err
	}

	cmd := b.Config.Cmd
	// set Cmd manually, this is special case only for Dockerfiles
	b.Config.Cmd = config.Cmd
	runconfig.Merge(b.Config, config)

	defer func(cmd []string) { b.Config.Cmd = cmd }(cmd)

	utils.Debugf("Command to be executed: %v", b.Config.Cmd)

	hit, err := b.probeCache()
	if err != nil {
		return err
	}
	if hit {
		return nil
	}

	c, err := b.create()
	if err != nil {
		return err
	}
	// Ensure that we keep the container mounted until the commit
	// to avoid unmounting and then mounting directly again
	c.Mount()
	defer c.Unmount()

	err = b.run(c)
	if err != nil {
		return err
	}
	if err := b.commit(c.ID, cmd, "run"); err != nil {
		return err
	}

	return nil
}

// CMD foo
//
// Set the default command to run in the container (which may be empty).
// Argument handling is the same as RUN.
//
func cmd(b *BuildFile, args []string) error {
	if len(args) < 2 {
		args = append([]string{"/bin/sh", "-c"}, args...)
	}

	b.Config.Cmd = args
	if err := b.commit("", b.Config.Cmd, fmt.Sprintf("CMD %v", cmd)); err != nil {
		return err
	}

	b.cmdSet = true
	return nil
}

// ENTRYPOINT /usr/sbin/nginx
//
// Set the entrypoint (which defaults to sh -c) to /usr/sbin/nginx. Will
// accept the CMD as the arguments to /usr/sbin/nginx.
//
// Handles command processing similar to CMD and RUN, only b.Config.Entrypoint
// is initialized at NewBuilder time instead of through argument parsing.
//
func entrypoint(b *BuildFile, args []string) error {
	b.Config.Entrypoint = args

	// if there is no cmd in current Dockerfile - cleanup cmd
	if !b.cmdSet {
		b.Config.Cmd = nil
	}
	if err := b.commit("", b.Config.Cmd, fmt.Sprintf("ENTRYPOINT %v", entrypoint)); err != nil {
		return err
	}
	return nil
}

// EXPOSE 6667/tcp 7000/tcp
//
// Expose ports for links and port mappings. This all ends up in
// b.Config.ExposedPorts for runconfig.
//
func expose(b *BuildFile, args []string) error {
	portsTab := args

	if b.Config.ExposedPorts == nil {
		b.Config.ExposedPorts = make(nat.PortSet)
	}

	ports, _, err := nat.ParsePortSpecs(append(portsTab, b.Config.PortSpecs...))
	if err != nil {
		return err
	}

	for port := range ports {
		if _, exists := b.Config.ExposedPorts[port]; !exists {
			b.Config.ExposedPorts[port] = struct{}{}
		}
	}
	b.Config.PortSpecs = nil

	return b.commit("", b.Config.Cmd, fmt.Sprintf("EXPOSE %v", ports))
}

// USER foo
//
// Set the user to 'foo' for future commands and when running the
// ENTRYPOINT/CMD at container run time.
//
func user(b *BuildFile, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("USER requires exactly one argument")
	}

	b.Config.User = args[0]
	return b.commit("", b.Config.Cmd, fmt.Sprintf("USER %v", args))
}

// VOLUME /foo
//
// Expose the volume /foo for use. Will also accept the JSON form, but either
// way requires exactly one argument.
//
func volume(b *BuildFile, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Volume cannot be empty")
	}

	volume := args

	if b.Config.Volumes == nil {
		b.Config.Volumes = map[string]struct{}{}
	}
	for _, v := range volume {
		b.Config.Volumes[v] = struct{}{}
	}
	if err := b.commit("", b.Config.Cmd, fmt.Sprintf("VOLUME %s", args)); err != nil {
		return err
	}
	return nil
}

// INSERT is no longer accepted, but we still parse it.
func insert(b *BuildFile, args []string) error {
	return fmt.Errorf("INSERT has been deprecated. Please use ADD instead")
}
