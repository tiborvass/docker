// Package builder is the evaluation step in the Dockerfile parse/evaluate pipeline.
//
// It incorporates a dispatch table based on the parser.Node values (see the
// parser package for more information) that are yielded from the parser itself.
// Calling NewBuilder with the BuildOpts struct can be used to customize the
// experience for execution purposes only. Parsing is controlled in the parser
// package, and this division of resposibility should be respected.
//
// Please see the jump table targets for the actual invocations, most of which
// will call out to the functions in internals.go to deal with their tasks.
//
// ONBUILD is a special case, which is covered in the onbuild() func in
// dispatchers.go.
//
// The evaluator uses the concept of "steps", which are usually each processable
// line in the Dockerfile. Each step is numbered and certain actions are taken
// before and after each step, such as creating an image ID and removing temporary
// containers and images. Note that ONBUILD creates a kinda-sorta "sub run" which
// includes its own set of steps (usually only one of them).
package dockerfile

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/docker/docker/builder/dockerfile/command"
	"github.com/docker/docker/builder/dockerfile/parser"
)

// Environment variable interpolation will happen on these statements only.
var replaceEnvAllowed = map[string]struct{}{
	command.Env:     {},
	command.Label:   {},
	command.Add:     {},
	command.Copy:    {},
	command.Workdir: {},
	command.Expose:  {},
	command.Volume:  {},
	command.User:    {},
}

var evaluateTable map[string]func(*Builder, []string, map[string]bool, string) error

func init() {
	evaluateTable = map[string]func(*Builder, []string, map[string]bool, string) error{
		command.Env:        env,
		command.Label:      label,
		command.Maintainer: maintainer,
		command.Add:        add,
		command.Copy:       dispatchCopy, // copy() is a go builtin
		command.From:       from,
		command.Onbuild:    onbuild,
		command.Workdir:    workdir,
		command.Run:        run,
		command.Cmd:        cmd,
		command.Entrypoint: entrypoint,
		command.Expose:     expose,
		command.Volume:     volume,
		command.User:       user,
	}
}

/*
// builder is an internal struct, used to maintain configuration of the Dockerfile's
// processing as it evaluates the parsing result.
type builder struct {
	Daemon *daemon.Daemon

	// effectively stdio for the run. Because it is not stdio, I said
	// "Effectively". Do not use stdio anywhere in this package for any reason.
	OutStream io.Writer
	ErrStream io.Writer

	Verbose      bool
	UtilizeCache bool
	cacheBusted  bool

	// controls how images and containers are handled between steps.
	Remove      bool
	ForceRemove bool
	Pull        bool

	// set this to true if we want the builder to not commit between steps.
	// This is useful when we only want to use the evaluator table to generate
	// the final configs of the Dockerfile but dont want the layers
	disableCommit bool

	// Registry server auth configs used to pull images when handling `FROM`.
	AuthConfigs map[string]cliconfig.AuthConfig

	// Deprecated, original writer used for ImagePull. To be removed.
	OutOld          io.Writer
	StreamFormatter *streamformatter.StreamFormatter

	Config *runconfig.Config // runconfig for cmd, run, entrypoint etc.

	// both of these are controlled by the Remove and ForceRemove options in BuildOpts
	TmpContainers map[string]struct{} // a map of containers used for removes

	dockerfileName string        // name of Dockerfile
	dockerfile     *parser.Node  // the syntax tree of the dockerfile
	image          string        // image name for commit processing
	maintainer     string        // maintainer name. could probably be removed.
	cmdSet         bool          // indicates is CMD was set in current Dockerfile
	BuilderFlags   *BFlags       // current cmd's BuilderFlags - temporary
	context        tarsum.TarSum // the context is a tarball that is uploaded by the client
	contextPath    string        // the path of the temporary directory the local context is unpacked to (server side)
	noBaseImage    bool          // indicates that this build does not start from any base image, but is being built from an empty file system.

	// Set resource restrictions for build containers
	cpuSetCpus   string
	cpuSetMems   string
	cpuShares    int64
	cpuPeriod    int64
	cpuQuota     int64
	cgroupParent string
	memory       int64
	memorySwap   int64
	ulimits      []*ulimit.Ulimit

	cancelled <-chan struct{} // When closed, job was cancelled.

	activeImages []string
	id           string // Used to hold reference images
}
*/

// This method is the entrypoint to all statement handling routines.
//
// Almost all nodes will have this structure:
// Child[Node, Node, Node] where Child is from parser.Node.Children and each
// node comes from parser.Node.Next. This forms a "line" with a statement and
// arguments and we process them in this normalized form by hitting
// evaluateTable with the leaf nodes of the command and the Builder object.
//
// ONBUILD is a special case; in this case the parser will emit:
// Child[Node, Child[Node, Node...]] where the first node is the literal
// "onbuild" and the child entrypoint is the command of the ONBUILD statmeent,
// such as `RUN` in ONBUILD RUN foo. There is special case logic in here to
// deal with that, at least until it becomes more of a general concern with new
// features.
func (b *Builder) dispatch(stepN int, ast *parser.Node) error {
	cmd := ast.Value
	upperCasedCmd := strings.ToUpper(cmd)

	// To ensure the user is give a decent error message if the platform
	// on which the daemon is running does not support a builder command.
	if err := platformSupports(strings.ToLower(cmd)); err != nil {
		return err
	}

	attrs := ast.Attributes
	original := ast.Original
	flags := ast.Flags
	strs := []string{}
	msg := fmt.Sprintf("Step %d : %s", stepN, upperCasedCmd)

	if len(ast.Flags) > 0 {
		msg += " " + strings.Join(ast.Flags, " ")
	}

	if cmd == "onbuild" {
		if ast.Next == nil {
			return fmt.Errorf("ONBUILD requires at least one argument")
		}
		ast = ast.Next.Children[0]
		strs = append(strs, ast.Value)
		msg += " " + ast.Value

		if len(ast.Flags) > 0 {
			msg += " " + strings.Join(ast.Flags, " ")
		}

	}

	// count the number of nodes that we are going to traverse first
	// so we can pre-create the argument and message array. This speeds up the
	// allocation of those list a lot when they have a lot of arguments
	cursor := ast
	var n int
	for cursor.Next != nil {
		cursor = cursor.Next
		n++
	}
	l := len(strs)
	strList := make([]string, n+l)
	copy(strList, strs)
	msgList := make([]string, n)

	var i int
	for ast.Next != nil {
		ast = ast.Next
		var str string
		str = ast.Value
		if _, ok := replaceEnvAllowed[cmd]; ok {
			var err error
			str, err = ProcessWord(ast.Value, b.runConfig.Env)
			if err != nil {
				return err
			}
		}
		strList[i+l] = str
		msgList[i] = ast.Value
		i++
	}

	msg += " " + strings.Join(msgList, " ")
	fmt.Fprintln(b.Stdout, msg)

	// XXX yes, we skip any cmds that are not valid; the parser should have
	// picked these out already.
	if f, ok := evaluateTable[cmd]; ok {
		b.flags = NewBFlags()
		b.flags.Args = flags
		return f(b, strList, attrs, original)
	}

	return fmt.Errorf("Unknown instruction: %s", upperCasedCmd)
}

// TODO: how to port this with client-side builder?
// platformSupports is a short-term function to give users a quality error
// message if a Dockerfile uses a command not supported on the platform.
func platformSupports(command string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	switch command {
	case "expose", "volume", "user":
		return fmt.Errorf("The daemon on this platform does not support the command '%s'", command)
	}
	return nil
}
