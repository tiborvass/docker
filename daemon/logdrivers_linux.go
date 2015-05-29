package daemon

// Importing packages here only to make sure their init gets called and
// therefore they register themselves to the logdriver factory.
import (
	_ "github.com/tiborvass/docker/daemon/logger/gelf"
	_ "github.com/tiborvass/docker/daemon/logger/journald"
	_ "github.com/tiborvass/docker/daemon/logger/jsonfilelog"
	_ "github.com/tiborvass/docker/daemon/logger/syslog"
)
