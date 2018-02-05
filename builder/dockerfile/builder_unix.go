// +build !windows

package dockerfile // import "github.com/tiborvass/docker/builder/dockerfile"

func defaultShellForOS(os string) []string {
	return []string{"/bin/sh", "-c"}
}
