package daemon

import (
	// Importing packages here only to make sure their init gets called and
	// therefore they register themselves to the logdriver factory.
	_ "github.com/tiborvass/docker/daemon/logger/awslogs"
	_ "github.com/tiborvass/docker/daemon/logger/etwlogs"
	_ "github.com/tiborvass/docker/daemon/logger/jsonfilelog"
	_ "github.com/tiborvass/docker/daemon/logger/splunk"
	_ "github.com/tiborvass/docker/daemon/logger/syslog"
)
