package signal // import "github.com/tiborvass/docker/pkg/signal"

import (
	"fmt"
	"os"
	gosignal "os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

// Trap sets up a simplified signal "trap", appropriate for common
// behavior expected from a vanilla unix command-line tool in general
// (and the Docker engine in particular).
//
// * If SIGINT or SIGTERM are received, `cleanup` is called, then the process is terminated.
// * If SIGINT or SIGTERM are received 3 times before cleanup is complete, then cleanup is
//   skipped and the process is terminated immediately (allows force quit of stuck daemon)
// * A SIGQUIT always causes an exit without cleanup, with a goroutine dump preceding exit.
// * Ignore SIGPIPE events. These are generated by systemd when journald is restarted while
//   the docker daemon is not restarted and also running under systemd.
//   Fixes https://github.com/docker/docker/issues/19728
//
func Trap(cleanup func(), logger interface {
	Info(args ...interface{})
}) {
	c := make(chan os.Signal, 1)
	// we will handle INT, TERM, QUIT, SIGPIPE here
	signals := []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGPIPE}
	gosignal.Notify(c, signals...)
	go func() {
		interruptCount := uint32(0)
		for sig := range c {
			if sig == syscall.SIGPIPE {
				continue
			}

			go func(sig os.Signal) {
				logger.Info(fmt.Sprintf("Processing signal '%v'", sig))
				switch sig {
				case os.Interrupt, syscall.SIGTERM:
					if atomic.LoadUint32(&interruptCount) < 3 {
						// Initiate the cleanup only once
						if atomic.AddUint32(&interruptCount, 1) == 1 {
							// Call the provided cleanup handler
							cleanup()
							os.Exit(0)
						} else {
							return
						}
					} else {
						// 3 SIGTERM/INT signals received; force exit without cleanup
						logger.Info("Forcing docker daemon shutdown without cleanup; 3 interrupts received")
					}
				case syscall.SIGQUIT:
					DumpStacks("")
					logger.Info("Forcing docker daemon shutdown without cleanup on SIGQUIT")
				}
				// for the SIGINT/TERM, and SIGQUIT non-clean shutdown case, exit with 128 + signal #
				os.Exit(128 + int(sig.(syscall.Signal)))
			}(sig)
		}
	}()
}

const stacksLogNameTemplate = "goroutine-stacks-%s.log"

// DumpStacks appends the runtime stack into file in dir and returns full path
// to that file.
func DumpStacks(dir string) (string, error) {
	var (
		buf       []byte
		stackSize int
	)
	bufferLen := 16384
	for stackSize == len(buf) {
		buf = make([]byte, bufferLen)
		stackSize = runtime.Stack(buf, true)
		bufferLen *= 2
	}
	buf = buf[:stackSize]
	var f *os.File
	if dir != "" {
		path := filepath.Join(dir, fmt.Sprintf(stacksLogNameTemplate, strings.Replace(time.Now().Format(time.RFC3339), ":", "", -1)))
		var err error
		f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return "", errors.Wrap(err, "failed to open file to write the goroutine stacks")
		}
		defer f.Close()
		defer f.Sync()
	} else {
		f = os.Stderr
	}
	if _, err := f.Write(buf); err != nil {
		return "", errors.Wrap(err, "failed to write goroutine stacks")
	}
	return f.Name(), nil
}
