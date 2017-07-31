package daemon

import (
	// Importing packages here only to make sure their init gets called and
	// therefore they register themselves to the logdriver factory.
	_ "github.com/moby/moby-core/daemon/logger/awslogs"
	_ "github.com/moby/moby-core/daemon/logger/etwlogs"
	_ "github.com/moby/moby-core/daemon/logger/fluentd"
	_ "github.com/moby/moby-core/daemon/logger/jsonfilelog"
	_ "github.com/moby/moby-core/daemon/logger/logentries"
	_ "github.com/moby/moby-core/daemon/logger/splunk"
	_ "github.com/moby/moby-core/daemon/logger/syslog"
)
