package dockerfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"

	"github.com/docker/docker/daemon"
	"github.com/docker/docker/pkg/httputils"
	"github.com/docker/docker/runconfig"
)

// When downloading remote contexts, limit the amount (in bytes)
// to be read from the response body in order to detect its Content-Type
const maxPreambleLength = 100

// whitelist of commands allowed for a commit/import
var validCommitCommands = map[string]bool{
	"cmd":        true,
	"entrypoint": true,
	"env":        true,
	"expose":     true,
	"label":      true,
	"onbuild":    true,
	"user":       true,
	"volume":     true,
	"workdir":    true,
}

// Build is the main interface of the package, it gathers the Builder
// struct and calls builder.Run() to do all the real build job.
func Build(d *daemon.Daemon, buildConfig *Config) error {
	return nil
}

// BuildFromConfig will do build directly from parameter 'changes', which comes
// from Dockerfile entries, it will:
//
// - call parse.Parse() to get AST root from Dockerfile entries
// - do build by calling builder.dispatch() to call all entries' handling routines
func BuildFromConfig(d *daemon.Daemon, c *runconfig.Config, changes []string) (*runconfig.Config, error) {
	/*
		ast, err := parser.Parse(bytes.NewBufferString(strings.Join(changes, "\n")))
		if err != nil {
			return nil, err
		}

		// ensure that the commands are valid
		for _, n := range ast.Children {
			if !validCommitCommands[n.Value] {
				return nil, fmt.Errorf("%s is not a valid change command", n.Value)
			}
		}

		builder := &builder{
			Daemon:        d,
			Config:        c,
			OutStream:     ioutil.Discard,
			ErrStream:     ioutil.Discard,
			disableCommit: true,
		}

		for i, n := range ast.Children {
			if err := builder.dispatch(i, n); err != nil {
				return nil, err
			}
		}

		return builder.Config, nil
	*/
	return nil, nil
}

// CommitConfig contains build configs for commit operation
type CommitConfig struct {
	Pause   bool
	Repo    string
	Tag     string
	Author  string
	Comment string
	Changes []string
	Config  *runconfig.Config
}

// Commit will create a new image from a container's changes
func Commit(name string, d *daemon.Daemon, c *CommitConfig) (string, error) {
	container, err := d.Get(name)
	if err != nil {
		return "", err
	}

	// It is not possible to commit a running container on Windows
	if runtime.GOOS == "windows" && container.IsRunning() {
		return "", fmt.Errorf("Windows does not support commit of a running container")
	}

	if c.Config == nil {
		c.Config = &runconfig.Config{}
	}

	newConfig, err := BuildFromConfig(d, c.Config, c.Changes)
	if err != nil {
		return "", err
	}

	if err := runconfig.Merge(newConfig, container.Config); err != nil {
		return "", err
	}

	commitCfg := &daemon.ContainerCommitConfig{
		Pause:   c.Pause,
		Repo:    c.Repo,
		Tag:     c.Tag,
		Author:  c.Author,
		Comment: c.Comment,
		Config:  newConfig,
	}

	img, err := d.Commit(container, commitCfg)
	if err != nil {
		return "", err
	}

	return img.ID, nil
}

// inspectResponse looks into the http response data at r to determine whether its
// content-type is on the list of acceptable content types for remote build contexts.
// This function returns:
//    - a string representation of the detected content-type
//    - an io.Reader for the response body
//    - an error value which will be non-nil either when something goes wrong while
//      reading bytes from r or when the detected content-type is not acceptable.
func inspectResponse(ct string, r io.ReadCloser, clen int64) (string, io.ReadCloser, error) {
	plen := clen
	if plen <= 0 || plen > maxPreambleLength {
		plen = maxPreambleLength
	}

	preamble := make([]byte, plen, plen)
	rlen, err := r.Read(preamble)
	if rlen == 0 {
		return ct, r, errors.New("Empty response")
	}
	if err != nil && err != io.EOF {
		return ct, r, err
	}

	preambleR := bytes.NewReader(preamble)
	bodyReader := ioutil.NopCloser(io.MultiReader(preambleR, r))
	// Some web servers will use application/octet-stream as the default
	// content type for files without an extension (e.g. 'Dockerfile')
	// so if we receive this value we better check for text content
	contentType := ct
	if len(ct) == 0 || ct == httputils.MimeTypes.OctetStream {
		contentType, _, err = httputils.DetectContentType(preamble)
		if err != nil {
			return contentType, bodyReader, err
		}
	}

	contentType = selectAcceptableMIME(contentType)
	var cterr error
	if len(contentType) == 0 {
		cterr = fmt.Errorf("unsupported Content-Type %q", ct)
		contentType = ct
	}

	return contentType, bodyReader, cterr
}
