package kernel // import "github.com/tiborvass/docker/pkg/parsers/kernel"

import (
	"golang.org/x/sys/unix"
)

func uname() (*unix.Utsname, error) {
	uts := &unix.Utsname{}

	if err := unix.Uname(uts); err != nil {
		return nil, err
	}
	return uts, nil
}
