package chrootarchive

import (
	"fmt"
	"os"

	"github.com/tiborvass/docker/pkg/reexec"
)

func init() {
	reexec.Register("docker-untar", untar)
	reexec.Register("docker-applyLayer", applyLayer)
	reexec.Init()
}

func fatal(err error) {
	fmt.Fprint(os.Stderr, err)
	os.Exit(1)
}
