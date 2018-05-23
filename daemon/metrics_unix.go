// +build !windows

package daemon // import "github.com/tiborvass/docker/daemon"

import (
	"net"
	"net/http"
	"path/filepath"

	"github.com/tiborvass/docker/pkg/plugingetter"
	"github.com/tiborvass/docker/pkg/plugins"
	"github.com/tiborvass/docker/plugin"
	"github.com/docker/go-metrics"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func (daemon *Daemon) listenMetricsSock() (string, error) {
	path := filepath.Join(daemon.configStore.ExecRoot, "metrics.sock")
	unix.Unlink(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		return "", errors.Wrap(err, "error setting up metrics plugin listener")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	go func() {
		http.Serve(l, mux)
	}()
	daemon.metricsPluginListener = l
	return path, nil
}

func registerMetricsPluginCallback(store *plugin.Store, sockPath string) {
	store.RegisterRuntimeOpt(metricsPluginType, func(s *specs.Spec) {
		f := plugin.WithSpecMounts([]specs.Mount{
			{Type: "bind", Source: sockPath, Destination: "/run/docker/metrics.sock", Options: []string{"bind", "ro"}},
		})
		f(s)
	})
	store.Handle(metricsPluginType, func(name string, client *plugins.Client) {
		// Use lookup since nothing in the system can really reference it, no need
		// to protect against removal
		p, err := store.Get(name, metricsPluginType, plugingetter.Lookup)
		if err != nil {
			return
		}

		if err := pluginStartMetricsCollection(p); err != nil {
			logrus.WithError(err).WithField("name", name).Error("error while initializing metrics plugin")
		}
	})
}
