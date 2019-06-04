// +build !windows

package chrootarchive // import "github.com/tiborvass/docker/pkg/chrootarchive"

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/tiborvass/docker/pkg/reexec"
)

func init() {
	reexec.Register("docker-applyLayer", applyLayer)
	reexec.Register("docker-untar", untar)
	reexec.Register("docker-tar", tar)
}

func fatal(err error) {
	fmt.Fprint(os.Stderr, err)
	os.Exit(1)
}

// flush consumes all the bytes from the reader discarding
// any errors
func flush(r io.Reader) (bytes int64, err error) {
	return io.Copy(ioutil.Discard, r)
}
