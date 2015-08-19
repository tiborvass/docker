package dockerfile

// This file contains the dispatchers for each command. Note that
// `nullDispatch` is not actually a command, but support for commands we parse
// but do nothing with.
//
// See evaluator.go for a higher level discussion of the whole evaluator
// package.

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/image"
	flag "github.com/docker/docker/pkg/mflag"
	"github.com/docker/docker/pkg/nat"
	"github.com/docker/docker/runconfig"
)

const (
	// NoBaseImageSpecifier is the symbol used by the FROM
	// command to specify that no base image is to be used.
	NoBaseImageSpecifier string = "scratch"
)

// dispatch with no layer / parsing. This is effectively not a command.
func nullDispatch(b *Builder, args []string, attributes map[string]bool, original string) error {
	return nil
}

// ENV foo bar
//
// Sets the environment variable foo to bar, also makes interpolation
// in the dockerfile available from the next statement on via ${foo}.
//
func env(b *Builder, args []string, attributes map[string]bool, original string) error {
	if len(args) == 0 {
		return fmt.Errorf("ENV requires at least one argument")
	}

	if len(args)%2 != 0 {
		// should never get here, but just in case
		return fmt.Errorf("Bad input to ENV, too many args")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	// TODO/FIXME/NOT USED
	// Just here to show how to use the builder flags stuff within the
	// context of a builder command. Will remove once we actually add
	// a builder command to something!
	/*
		flBool1 := b.flags.AddBool("bool1", false)
		flStr1 := b.flags.AddString("str1", "HI")

		if err := b.flags.Parse(); err != nil {
			return err
		}

		fmt.Printf("Bool1:%v\n", flBool1)
		fmt.Printf("Str1:%v\n", flStr1)
	*/

	commitStr := "ENV"

	for j := 0; j < len(args); j++ {
		// name  ==> args[j]
		// value ==> args[j+1]
		newVar := args[j] + "=" + args[j+1] + ""
		commitStr += " " + newVar

		gotOne := false
		for i, envVar := range b.runConfig.Env {
			envParts := strings.SplitN(envVar, "=", 2)
			if envParts[0] == args[j] {
				b.runConfig.Env[i] = newVar
				gotOne = true
				break
			}
		}
		if !gotOne {
			b.runConfig.Env = append(b.runConfig.Env, newVar)
		}
		j++
	}

	return b.commit("", b.runConfig.Cmd, commitStr)
}

// MAINTAINER some text <maybe@an.email.address>
//
// Sets the maintainer metadata.
func maintainer(b *Builder, args []string, attributes map[string]bool, original string) error {
	if len(args) != 1 {
		return fmt.Errorf("MAINTAINER requires exactly one argument")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	b.maintainer = args[0]
	return b.commit("", b.runConfig.Cmd, fmt.Sprintf("MAINTAINER %s", b.maintainer))
}

// LABEL some json data describing the image
//
// Sets the Label variable foo to bar,
//
func label(b *Builder, args []string, attributes map[string]bool, original string) error {
	if len(args) == 0 {
		return fmt.Errorf("LABEL requires at least one argument")
	}
	if len(args)%2 != 0 {
		// should never get here, but just in case
		return fmt.Errorf("Bad input to LABEL, too many args")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	commitStr := "LABEL"

	if b.runConfig.Labels == nil {
		b.runConfig.Labels = map[string]string{}
	}

	for j := 0; j < len(args); j++ {
		// name  ==> args[j]
		// value ==> args[j+1]
		newVar := args[j] + "=" + args[j+1] + ""
		commitStr += " " + newVar

		b.runConfig.Labels[args[j]] = args[j+1]
		j++
	}
	return b.commit("", b.runConfig.Cmd, commitStr)
}

// ADD foo /path
//
// Add the file 'foo' to '/path'. Tarball and Remote URL (git, http) handling
// exist here. If you do not wish to have this automatic handling, use COPY.
//
func add(b *Builder, args []string, attributes map[string]bool, original string) error {
	if len(args) < 2 {
		return fmt.Errorf("ADD requires at least two arguments")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	return b.runContextCommand(args, true, true, "ADD")
}

// COPY foo /path
//
// Same as 'ADD' but without the tar and remote url handling.
//
func dispatchCopy(b *Builder, args []string, attributes map[string]bool, original string) error {
	if len(args) < 2 {
		return fmt.Errorf("COPY requires at least two arguments")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	return b.runContextCommand(args, false, false, "COPY")
}

// FROM imagename
//
// This sets the image the dockerfile will build on top of.
//
func from(b *Builder, args []string, attributes map[string]bool, original string) error {
	if len(args) != 1 {
		return fmt.Errorf("FROM requires one argument")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	name := args[0]

	// Windows cannot support a container with no base image.
	if name == NoBaseImageSpecifier {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("Windows does not support FROM scratch")
		}
		b.image = ""
		b.noBaseImage = true
		return nil
	}

	var (
		image      *image.Image
		err        error
		shouldPull = b.Pull
	)
	// TODO: don't use `name`, instead resolve it to a digest
	if !shouldPull {
		//image, err := b.Daemon.Repositories().LookupImage(name)
		image, err = b.docker.LookupImage(name)
		// TODO: shouldn't we error out if error is different from "not found" ?
		if image == nil {
			shouldPull = true
		}
	}
	if shouldPull {
		//image, err := b.pullImage(name)
		image, err = b.docker.Pull(name)
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
func onbuild(b *Builder, args []string, attributes map[string]bool, original string) error {
	if len(args) == 0 {
		return fmt.Errorf("ONBUILD requires at least one argument")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	triggerInstruction := strings.ToUpper(strings.TrimSpace(args[0]))
	switch triggerInstruction {
	case "ONBUILD":
		return fmt.Errorf("Chaining ONBUILD via `ONBUILD ONBUILD` isn't allowed")
	case "MAINTAINER", "FROM":
		return fmt.Errorf("%s isn't allowed as an ONBUILD trigger", triggerInstruction)
	}

	original = regexp.MustCompile(`(?i)^\s*ONBUILD\s*`).ReplaceAllString(original, "")

	b.runConfig.OnBuild = append(b.runConfig.OnBuild, original)
	return b.commit("", b.runConfig.Cmd, fmt.Sprintf("ONBUILD %s", original))
}

// WORKDIR /tmp
//
// Set the working directory for future RUN/CMD/etc statements.
//
func workdir(b *Builder, args []string, attributes map[string]bool, original string) error {
	if len(args) != 1 {
		return fmt.Errorf("WORKDIR requires exactly one argument")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	// Note that workdir passed comes from the Dockerfile. Hence it is in
	// Linux format using forward-slashes, even on Windows. However,
	// b.runConfig.WorkingDir is in platform-specific notation (in other words
	// on Windows will use `\`
	workdir := args[0]

	isAbs := false
	if runtime.GOOS == "windows" {
		// Alternate processing for Windows here is necessary as we can't call
		// filepath.IsAbs(workDir) as that would verify Windows style paths,
		// along with drive-letters (eg c:\pathto\file.txt). We (arguably
		// correctly or not) check for both forward and back slashes as this
		// is what the 1.4.2 GoLang implementation of IsAbs() does in the
		// isSlash() function.
		isAbs = workdir[0] == '\\' || workdir[0] == '/'
	} else {
		isAbs = filepath.IsAbs(workdir)
	}

	if !isAbs {
		current := b.runConfig.WorkingDir
		if runtime.GOOS == "windows" {
			// Convert to Linux format before join
			current = strings.Replace(current, "\\", "/", -1)
		}
		// Must use path.Join so works correctly on Windows, not filepath
		workdir = path.Join("/", current, workdir)
	}

	// Convert to platform specific format
	if runtime.GOOS == "windows" {
		workdir = strings.Replace(workdir, "/", "\\", -1)
	}
	b.runConfig.WorkingDir = workdir

	return b.commit("", b.runConfig.Cmd, fmt.Sprintf("WORKDIR %v", workdir))
}

// RUN some command yo
//
// run a command and commit the image. Args are automatically prepended with
// 'sh -c' under linux or 'cmd /S /C' under Windows, in the event there is
// only one argument. The difference in processing:
//
// RUN echo hi          # sh -c echo hi       (Linux)
// RUN echo hi          # cmd /S /C echo hi   (Windows)
// RUN [ "echo", "hi" ] # echo hi
//
func run(b *Builder, args []string, attributes map[string]bool, original string) error {
	if b.image == "" && !b.noBaseImage {
		return fmt.Errorf("Please provide a source image with `from` prior to run")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	args = handleJSONArgs(args, attributes)

	if !attributes["json"] {
		if runtime.GOOS != "windows" {
			args = append([]string{"/bin/sh", "-c"}, args...)
		} else {
			args = append([]string{"cmd", "/S /C"}, args...)
		}
	}

	runCmd := flag.NewFlagSet("run", flag.ContinueOnError)
	runCmd.SetOutput(ioutil.Discard)
	runCmd.Usage = nil

	config, _, _, err := runconfig.Parse(runCmd, append([]string{b.image}, args...))
	if err != nil {
		return err
	}

	cmd := b.runConfig.Cmd
	// set Cmd manually, this is special case only for Dockerfiles
	b.runConfig.Cmd = config.Cmd
	runconfig.Merge(b.runConfig, config)

	defer func(cmd *runconfig.Command) { b.runConfig.Cmd = cmd }(cmd)

	logrus.Debugf("[BUILDER] Command to be executed: %v", b.runConfig.Cmd)

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
func cmd(b *Builder, args []string, attributes map[string]bool, original string) error {
	if err := b.flags.Parse(); err != nil {
		return err
	}

	cmdSlice := handleJSONArgs(args, attributes)

	if !attributes["json"] {
		if runtime.GOOS != "windows" {
			cmdSlice = append([]string{"/bin/sh", "-c"}, cmdSlice...)
		} else {
			cmdSlice = append([]string{"cmd", "/S /C"}, cmdSlice...)
		}
	}

	b.runConfig.Cmd = runconfig.NewCommand(cmdSlice...)

	if err := b.commit("", b.runConfig.Cmd, fmt.Sprintf("CMD %q", cmdSlice)); err != nil {
		return err
	}

	if len(args) != 0 {
		b.cmdSet = true
	}

	return nil
}

// ENTRYPOINT /usr/sbin/nginx
//
// Set the entrypoint (which defaults to sh -c on linux, or cmd /S /C on Windows) to
// /usr/sbin/nginx. Will accept the CMD as the arguments to /usr/sbin/nginx.
//
// Handles command processing similar to CMD and RUN, only b.runConfig.Entrypoint
// is initialized at NewBuilder time instead of through argument parsing.
//
func entrypoint(b *Builder, args []string, attributes map[string]bool, original string) error {
	if err := b.flags.Parse(); err != nil {
		return err
	}

	parsed := handleJSONArgs(args, attributes)

	switch {
	case attributes["json"]:
		// ENTRYPOINT ["echo", "hi"]
		b.runConfig.Entrypoint = runconfig.NewEntrypoint(parsed...)
	case len(parsed) == 0:
		// ENTRYPOINT []
		b.runConfig.Entrypoint = nil
	default:
		// ENTRYPOINT echo hi
		if runtime.GOOS != "windows" {
			b.runConfig.Entrypoint = runconfig.NewEntrypoint("/bin/sh", "-c", parsed[0])
		} else {
			b.runConfig.Entrypoint = runconfig.NewEntrypoint("cmd", "/S /C", parsed[0])
		}
	}

	// when setting the entrypoint if a CMD was not explicitly set then
	// set the command to nil
	if !b.cmdSet {
		b.runConfig.Cmd = nil
	}

	if err := b.commit("", b.runConfig.Cmd, fmt.Sprintf("ENTRYPOINT %q", b.runConfig.Entrypoint)); err != nil {
		return err
	}

	return nil
}

// EXPOSE 6667/tcp 7000/tcp
//
// Expose ports for links and port mappings. This all ends up in
// b.runConfig.ExposedPorts for runconfig.
//
func expose(b *Builder, args []string, attributes map[string]bool, original string) error {
	portsTab := args

	if len(args) == 0 {
		return fmt.Errorf("EXPOSE requires at least one argument")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	if b.runConfig.ExposedPorts == nil {
		b.runConfig.ExposedPorts = make(nat.PortSet)
	}

	ports, _, err := nat.ParsePortSpecs(portsTab)
	if err != nil {
		return err
	}

	// instead of using ports directly, we build a list of ports and sort it so
	// the order is consistent. This prevents cache burst where map ordering
	// changes between builds
	portList := make([]string, len(ports))
	var i int
	for port := range ports {
		if _, exists := b.runConfig.ExposedPorts[port]; !exists {
			b.runConfig.ExposedPorts[port] = struct{}{}
		}
		portList[i] = string(port)
		i++
	}
	sort.Strings(portList)
	return b.commit("", b.runConfig.Cmd, fmt.Sprintf("EXPOSE %s", strings.Join(portList, " ")))
}

// USER foo
//
// Set the user to 'foo' for future commands and when running the
// ENTRYPOINT/CMD at container run time.
//
func user(b *Builder, args []string, attributes map[string]bool, original string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("USER is not supported on Windows")
	}

	if len(args) != 1 {
		return fmt.Errorf("USER requires exactly one argument")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	b.runConfig.User = args[0]
	return b.commit("", b.runConfig.Cmd, fmt.Sprintf("USER %v", args))
}

// VOLUME /foo
//
// Expose the volume /foo for use. Will also accept the JSON array form.
//
func volume(b *Builder, args []string, attributes map[string]bool, original string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("VOLUME is not supported on Windows")
	}
	if len(args) == 0 {
		return fmt.Errorf("VOLUME requires at least one argument")
	}

	if err := b.flags.Parse(); err != nil {
		return err
	}

	if b.runConfig.Volumes == nil {
		b.runConfig.Volumes = map[string]struct{}{}
	}
	for _, v := range args {
		v = strings.TrimSpace(v)
		if v == "" {
			return fmt.Errorf("Volume specified can not be an empty string")
		}
		b.runConfig.Volumes[v] = struct{}{}
	}
	if err := b.commit("", b.runConfig.Cmd, fmt.Sprintf("VOLUME %v", args)); err != nil {
		return err
	}
	return nil
}
