// +build !linux

package native

import (
	"fmt"

	"github.com/tiborvass/docker/daemon/execdriver"
)

func NewDriver(root, initPath string) (execdriver.Driver, error) {
	return nil, fmt.Errorf("native driver not supported on non-linux")
}
