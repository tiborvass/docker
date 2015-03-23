package syslog

import (
	"fmt"
	"log/syslog"
	"os"
	"path"
	"sync"

	"github.com/tiborvass/docker/daemon/logger"
)

type Syslog struct {
	writer *syslog.Writer
	tag    string
	mu     sync.Mutex
}

func New(tag string) (logger.Logger, error) {
	log, err := syslog.New(syslog.LOG_USER, path.Base(os.Args[0]))
	if err != nil {
		return nil, err
	}
	return &Syslog{
		writer: log,
		tag:    tag,
	}, nil
}

func (s *Syslog) Log(msg *logger.Message) error {
	logMessage := fmt.Sprintf("%s: %s", s.tag, string(msg.Line))
	if msg.Source == "stderr" {
		if err := s.writer.Err(logMessage); err != nil {
			return err
		}

	} else {
		if err := s.writer.Info(logMessage); err != nil {
			return err
		}
	}
	return nil
}

func (s *Syslog) Close() error {
	if s.writer != nil {
		return s.writer.Close()
	}
	return nil
}

func (s *Syslog) Name() string {
	return "Syslog"
}
