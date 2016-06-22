// +build !windows

package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/pkg/testutil/assert"
	"github.com/tiborvass/docker/pkg/testutil/tempfile"
)

func TestLoadDaemonCliConfigWithDaemonFlags(t *testing.T) {
	content := `{"log-opts": {"max-size": "1k"}}`
	tempFile := tempfile.NewTempFile(t, "config", content)
	defer tempFile.Remove()

	opts := defaultOptions(tempFile.Name())
	opts.common.Debug = true
	opts.common.LogLevel = "info"
	assert.NilError(t, opts.flags.Set("selinux-enabled", "true"))

	loadedConfig, err := loadDaemonCliConfig(opts)
	assert.NilError(t, err)
	assert.NotNil(t, loadedConfig)

	assert.Equal(t, loadedConfig.Debug, true)
	assert.Equal(t, loadedConfig.LogLevel, "info")
	assert.Equal(t, loadedConfig.EnableSelinuxSupport, true)
	assert.Equal(t, loadedConfig.LogConfig.Type, "json-file")
	assert.Equal(t, loadedConfig.LogConfig.Config["max-size"], "1k")
}

func TestLoadDaemonConfigWithNetwork(t *testing.T) {
	content := `{"bip": "127.0.0.2", "ip": "127.0.0.1"}`
	tempFile := tempfile.NewTempFile(t, "config", content)
	defer tempFile.Remove()

	opts := defaultOptions(tempFile.Name())
	loadedConfig, err := loadDaemonCliConfig(opts)
	assert.NilError(t, err)
	assert.NotNil(t, loadedConfig)

	assert.Equal(t, loadedConfig.IP, "127.0.0.2")
	assert.Equal(t, loadedConfig.DefaultIP.String(), "127.0.0.1")
}

func TestLoadDaemonConfigWithMapOptions(t *testing.T) {
	content := `{
		"cluster-store-opts": {"kv.cacertfile": "/var/lib/docker/discovery_certs/ca.pem"},
		"log-opts": {"tag": "test"}
}`
	tempFile := tempfile.NewTempFile(t, "config", content)
	defer tempFile.Remove()

	opts := defaultOptions(tempFile.Name())
	loadedConfig, err := loadDaemonCliConfig(opts)
	assert.NilError(t, err)
	assert.NotNil(t, loadedConfig)
	assert.NotNil(t, loadedConfig.ClusterOpts)

	expectedPath := "/var/lib/docker/discovery_certs/ca.pem"
	assert.Equal(t, loadedConfig.ClusterOpts["kv.cacertfile"], expectedPath)
	assert.NotNil(t, loadedConfig.LogConfig.Config)
	assert.Equal(t, loadedConfig.LogConfig.Config["tag"], "test")
}

func TestLoadDaemonConfigWithTrueDefaultValues(t *testing.T) {
	content := `{ "userland-proxy": false }`
	tempFile := tempfile.NewTempFile(t, "config", content)
	defer tempFile.Remove()

	opts := defaultOptions(tempFile.Name())
	loadedConfig, err := loadDaemonCliConfig(opts)
	assert.NilError(t, err)
	assert.NotNil(t, loadedConfig)
	assert.NotNil(t, loadedConfig.ClusterOpts)

	assert.Equal(t, loadedConfig.EnableUserlandProxy, false)

	// make sure reloading doesn't generate configuration
	// conflicts after normalizing boolean values.
	reload := func(reloadedConfig *daemon.Config) {
		assert.Equal(t, reloadedConfig.EnableUserlandProxy, false)
	}
	assert.NilError(t, daemon.ReloadConfiguration(opts.configFile, opts.flags, reload))
}

func TestLoadDaemonConfigWithTrueDefaultValuesLeaveDefaults(t *testing.T) {
	tempFile := tempfile.NewTempFile(t, "config", `{}`)
	defer tempFile.Remove()

	opts := defaultOptions(tempFile.Name())
	loadedConfig, err := loadDaemonCliConfig(opts)
	assert.NilError(t, err)
	assert.NotNil(t, loadedConfig)
	assert.NotNil(t, loadedConfig.ClusterOpts)

	assert.Equal(t, loadedConfig.EnableUserlandProxy, true)
}

func TestLoadDaemonConfigWithLegacyRegistryOptions(t *testing.T) {
	c := &daemon.Config{}
	common := &cliflags.CommonFlags{}
	flags := mflag.NewFlagSet("test", mflag.ContinueOnError)
	c.ServiceOptions.InstallCliFlags(flags, absentFromHelp)

	f, err := ioutil.TempFile("", "docker-config-")
	if err != nil {
		t.Fatal(err)
	}
	configFile := f.Name()
	defer os.Remove(configFile)

	f.Write([]byte(`{"disable-legacy-registry": true}`))
	f.Close()

	loadedConfig, err := loadDaemonCliConfig(c, flags, common, configFile)
	if err != nil {
		t.Fatal(err)
	}
	if loadedConfig == nil {
		t.Fatal("expected configuration, got nil")
	}

	if !loadedConfig.V2Only {
		t.Fatal("expected disable-legacy-registry to be true, got false")
	}
}
