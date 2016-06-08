// +build experimental

package stack

import (
	"fmt"
	"io"
	"os"

	"github.com/tiborvass/docker/api/client/bundlefile"
	"github.com/spf13/pflag"
)

func addBundlefileFlag(opt *string, flags *pflag.FlagSet) {
	flags.StringVarP(
		opt,
		"bundle", "f", "",
		"Path to a bundle (Default: STACK.dsb)")
}

func loadBundlefile(stderr io.Writer, namespace string, path string) (*bundlefile.Bundlefile, error) {
	defaultPath := fmt.Sprintf("%s.dsb", namespace)

	if path == "" {
		path = defaultPath
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf(
			"Bundle %s not found. Specify the path with -f or --bundle",
			path)
	}

	fmt.Fprintf(stderr, "Loading bundle from %s\n", path)
	bundle, err := bundlefile.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s: %v\n", path, err)
	}
	return bundle, err
}
