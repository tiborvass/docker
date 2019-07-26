package build // import "github.com/docker/docker/integration-cli/cli/build"

import (
	"io"
	"strings"
	"testing"

	"github.com/docker/docker/testutil/fakecontext"
	"github.com/docker/docker/builder/buildutil"
	"gotest.tools/icmd"
)

// WithStdinContext sets the build context from the standard input with the specified reader
func WithStdinContext(closer io.ReadCloser) func(*icmd.Cmd) func() {
	return func(cmd *icmd.Cmd) func() {
		cmd.Command = append(cmd.Command, "-")
		cmd.Stdin = closer
		return func() {
			// FIXME(vdemeester) we should not ignore the error here…
			closer.Close()
		}
	}
}

// WithTarget sets the build target
func WithTarget(target string) func(*icmd.Cmd) func() {
	return func(cmd *icmd.Cmd) func() {
		cmd.Command = append(cmd.Command, "--target", target)
		return nil
	}
}

// WithDockerfile creates / returns a CmdOperator to set the Dockerfile for a build operation
func WithDockerfile(dockerfile string) func(*icmd.Cmd) func() {
	return func(cmd *icmd.Cmd) func() {
		cmd.Command = append(cmd.Command, "-")
		cmd.Stdin = strings.NewReader(dockerfile)
		return nil
	}
}

// WithoutCache makes the build ignore cache
func WithoutCache(cmd *icmd.Cmd) func() {
	cmd.Command = append(cmd.Command, "--no-cache")
	return nil
}

// WithContextPath sets the build context path
func WithContextPath(path string) func(*icmd.Cmd) func() {
	return func(cmd *icmd.Cmd) func() {
		cmd.Command = append(cmd.Command, path)
		return nil
	}
}

// WithExternalBuildContext use the specified context as build context
func WithExternalBuildContext(ctx *fakecontext.Fake) func(*icmd.Cmd) func() {
	return func(cmd *icmd.Cmd) func() {
		cmd.Dir = ctx.Dir
		cmd.Command = append(cmd.Command, ".")
		return nil
	}
}

// WithBuildContext sets up the build context
func WithBuildContext(t testing.TB, contextOperators ...func(*fakecontext.Fake) error) func(*icmd.Cmd) func() {
	// FIXME(vdemeester) de-duplicate that
	ctx := fakecontext.New(t, "", contextOperators...)
	return func(cmd *icmd.Cmd) func() {
		cmd.Dir = ctx.Dir
		cmd.Command = append(cmd.Command, ".")
		return closeBuildContext(t, ctx)
	}
}

// WithFile adds the specified file (with content) in the build context
func WithFile(name, content string) func(*fakecontext.Fake) error {
	return fakecontext.WithFile(name, content)
}

func Expect(out string, exitcode int) icmd.Expected {
	if buildutil.BuildKitEnabled() {
		if exitcode != 0 {
			exitcode = 1
		}
		return icmd.Expected{Err: out, ExitCode: exitcode}
	}
	return icmd.Expected{Out: out, ExitCode: exitcode}
}

func closeBuildContext(t testing.TB, ctx *fakecontext.Fake) func() {
	return func() {
		if err := ctx.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
