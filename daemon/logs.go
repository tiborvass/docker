package daemon

import (
	"io"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/container"
	"github.com/tiborvass/docker/daemon/logger"
	"github.com/tiborvass/docker/daemon/logger/jsonfilelog"
	derr "github.com/tiborvass/docker/errors"
	"github.com/tiborvass/docker/pkg/stdcopy"
	"github.com/docker/engine-api/types/backend"
)

// ContainerLogs hooks up a container's stdout and stderr streams
// configured with the given struct.
func (daemon *Daemon) ContainerLogs(containerName string, config *backend.ContainerLogsConfig) error {
	container, err := daemon.GetContainer(containerName)
	if err != nil {
		return derr.ErrorCodeNoSuchContainer.WithArgs(containerName)
	}

	if !(config.UseStdout || config.UseStderr) {
		return derr.ErrorCodeNeedStream
	}

	outStream := config.OutStream
	errStream := outStream
	if !container.Config.Tty {
		errStream = stdcopy.NewStdWriter(outStream, stdcopy.Stderr)
		outStream = stdcopy.NewStdWriter(outStream, stdcopy.Stdout)
	}
	config.OutStream = outStream

	cLog, err := daemon.getLogger(container)
	if err != nil {
		return err
	}
	logReader, ok := cLog.(logger.LogReader)
	if !ok {
		return logger.ErrReadLogsNotSupported
	}

	follow := config.Follow && container.IsRunning()
	tailLines, err := strconv.Atoi(config.Tail)
	if err != nil {
		tailLines = -1
	}

	logrus.Debug("logs: begin stream")
	readConfig := logger.ReadConfig{
		Since:  config.Since,
		Tail:   tailLines,
		Follow: follow,
	}
	logs := logReader.ReadLogs(readConfig)

	for {
		select {
		case err := <-logs.Err:
			logrus.Errorf("Error streaming logs: %v", err)
			return nil
		case <-config.Stop:
			logs.Close()
			return nil
		case msg, ok := <-logs.Msg:
			if !ok {
				logrus.Debugf("logs: end stream")
				return nil
			}
			logLine := msg.Line
			if config.Timestamps {
				logLine = append([]byte(msg.Timestamp.Format(logger.TimeFormat)+" "), logLine...)
			}
			if msg.Source == "stdout" && config.UseStdout {
				outStream.Write(logLine)
			}
			if msg.Source == "stderr" && config.UseStderr {
				errStream.Write(logLine)
			}
		}
	}
}

func (daemon *Daemon) getLogger(container *container.Container) (logger.Logger, error) {
	if container.LogDriver != nil && container.IsRunning() {
		return container.LogDriver, nil
	}
	cfg := container.GetLogConfig(daemon.defaultLogConfig)
	if err := logger.ValidateLogOpts(cfg.Type, cfg.Config); err != nil {
		return nil, err
	}
	return container.StartLogger(cfg)
}

// StartLogging initializes and starts the container logging stream.
func (daemon *Daemon) StartLogging(container *container.Container) error {
	cfg := container.GetLogConfig(daemon.defaultLogConfig)
	if cfg.Type == "none" {
		return nil // do not start logging routines
	}

	if err := logger.ValidateLogOpts(cfg.Type, cfg.Config); err != nil {
		return err
	}
	l, err := container.StartLogger(cfg)
	if err != nil {
		return derr.ErrorCodeInitLogger.WithArgs(err)
	}

	copier := logger.NewCopier(container.ID, map[string]io.Reader{"stdout": container.StdoutPipe(), "stderr": container.StderrPipe()}, l)
	container.LogCopier = copier
	copier.Run()
	container.LogDriver = l

	// set LogPath field only for json-file logdriver
	if jl, ok := l.(*jsonfilelog.JSONFileLogger); ok {
		container.LogPath = jl.LogPath()
	}

	return nil
}
