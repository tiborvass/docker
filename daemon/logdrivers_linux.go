package daemon // import "github.com/tiborvass/docker/daemon"

import (
	// Importing packages here only to make sure their init gets called and
	// therefore they register themselves to the logdriver factory.
	_ "github.com/tiborvass/docker/daemon/logger/awslogs"
	_ "github.com/tiborvass/docker/daemon/logger/fluentd"
	_ "github.com/tiborvass/docker/daemon/logger/gcplogs"
	_ "github.com/tiborvass/docker/daemon/logger/gelf"
	_ "github.com/tiborvass/docker/daemon/logger/journald"
	_ "github.com/tiborvass/docker/daemon/logger/jsonfilelog"
	_ "github.com/tiborvass/docker/daemon/logger/local"
	_ "github.com/tiborvass/docker/daemon/logger/logentries"
	_ "github.com/tiborvass/docker/daemon/logger/splunk"
	_ "github.com/tiborvass/docker/daemon/logger/syslog"
)
