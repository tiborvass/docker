// +build windows

package config // import "github.com/tiborvass/docker/daemon/config"

import (
	"io/ioutil"
	"testing"

	"github.com/tiborvass/docker/opts"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/spf13/pflag"
)

func TestDaemonConfigurationMerge(t *testing.T) {
	f, err := ioutil.TempFile("", "docker-config-")
	if err != nil {
		t.Fatal(err)
	}

	configFile := f.Name()

	f.Write([]byte(`
		{
			"debug": true,
			"log-opts": {
				"tag": "test_tag"
			}
		}`))

	f.Close()

	c := &Config{
		CommonConfig: CommonConfig{
			AutoRestart: true,
			LogConfig: LogConfig{
				Type:   "syslog",
				Config: map[string]string{"tag": "test"},
			},
		},
	}

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	var debug bool
	flags.BoolVarP(&debug, "debug", "D", false, "")
	flags.Var(opts.NewNamedMapOpts("log-opts", nil, nil), "log-opt", "")

	cc, err := MergeDaemonConfigurations(c, flags, configFile)
	assert.NilError(t, err)

	assert.Check(t, cc.Debug)
	assert.Check(t, cc.AutoRestart)

	expectedLogConfig := LogConfig{
		Type:   "syslog",
		Config: map[string]string{"tag": "test_tag"},
	}

	assert.Check(t, is.DeepEqual(expectedLogConfig, cc.LogConfig))
}
