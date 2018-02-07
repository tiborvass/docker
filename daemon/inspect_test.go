package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"testing"

	containertypes "github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/daemon/config"
	"github.com/tiborvass/docker/daemon/exec"

	"github.com/stretchr/testify/assert"
)

func TestGetInspectData(t *testing.T) {
	c := &container.Container{
		ID:           "inspect-me",
		HostConfig:   &containertypes.HostConfig{},
		State:        container.NewState(),
		ExecCommands: exec.NewStore(),
	}

	d := &Daemon{
		linkIndex:   newLinkIndex(),
		configStore: &config.Config{},
	}

	_, err := d.getInspectData(c)
	assert.Error(t, err)

	c.Dead = true
	_, err = d.getInspectData(c)
	assert.NoError(t, err)
}
