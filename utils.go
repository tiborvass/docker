package docker

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"github.com/dotcloud/docker/rcli"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Go is a basic promise implementation: it wraps calls a function in a goroutine,
// and returns a channel which will later return the function's return value.
func Go(f func() error) chan error {
	ch := make(chan error)
	go func() {
		ch <- f()
	}()
	return ch
}

// Request a given URL and return an io.Reader
func Download(url string, stderr io.Writer) (*http.Response, error) {
	var resp *http.Response
	var err error = nil
	if resp, err = http.Get(url); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, errors.New("Got HTTP status code >= 400: " + resp.Status)
	}
	return resp, nil
}

// Debug function, if the debug flag is set, then display. Do nothing otherwise
// If Docker is in damon mode, also send the debug info on the socket
func Debugf(format string, a ...interface{}) {
	if rcli.DEBUG_FLAG {

		// Retrieve the stack infos
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "<unknown>"
			line = -1
		} else {
			file = file[strings.LastIndex(file, "/")+1:]
		}

		fmt.Fprintf(os.Stderr, fmt.Sprintf("[debug] %s:%d %s\n", file, line, format), a...)
		if rcli.CLIENT_SOCKET != nil {
			fmt.Fprintf(rcli.CLIENT_SOCKET, fmt.Sprintf("[debug] %s:%d %s\n", file, line, format), a...)
		}
	}
}

// Reader with progress bar
type progressReader struct {
	reader        io.ReadCloser // Stream to read from
	output        io.Writer     // Where to send progress bar to
	read_total    int           // Expected stream length (bytes)
	read_progress int           // How much has been read so far (bytes)
	last_update   int           // How many bytes read at least update
}

func (r *progressReader) Read(p []byte) (n int, err error) {
	read, err := io.ReadCloser(r.reader).Read(p)
	r.read_progress += read

	// Only update progress for every 1% read
	update_every := int(0.01 * float64(r.read_total))
	if r.read_progress-r.last_update > update_every || r.read_progress == r.read_total {
		fmt.Fprintf(r.output, "%d/%d (%.0f%%)\r",
			r.read_progress,
			r.read_total,
			float64(r.read_progress)/float64(r.read_total)*100)
		r.last_update = r.read_progress
	}
	// Send newline when complete
	if err == io.EOF {
		fmt.Fprintf(r.output, "\n")
	}

	return read, err
}
func (r *progressReader) Close() error {
	return io.ReadCloser(r.reader).Close()
}
func ProgressReader(r io.ReadCloser, size int, output io.Writer) *progressReader {
	return &progressReader{r, output, size, 0, 0}
}

// HumanDuration returns a human-readable approximation of a duration
// (eg. "About a minute", "4 hours ago", etc.)
func HumanDuration(d time.Duration) string {
	if seconds := int(d.Seconds()); seconds < 1 {
		return "Less than a second"
	} else if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	} else if minutes := int(d.Minutes()); minutes == 1 {
		return "About a minute"
	} else if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if hours := int(d.Hours()); hours == 1 {
		return "About an hour"
	} else if hours < 48 {
		return fmt.Sprintf("%d hours", hours)
	} else if hours < 24*7*2 {
		return fmt.Sprintf("%d days", hours/24)
	} else if hours < 24*30*3 {
		return fmt.Sprintf("%d weeks", hours/24/7)
	} else if hours < 24*365*2 {
		return fmt.Sprintf("%d months", hours/24/30)
	}
	return fmt.Sprintf("%d years", d.Hours()/24/365)
}

func Trunc(s string, maxlen int) string {
	if len(s) <= maxlen {
		return s
	}
	return s[:maxlen]
}

// Figure out the absolute path of our own binary
func SelfPath() string {
	path, err := exec.LookPath(os.Args[0])
	if err != nil {
		panic(err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return path
}

type nopWriteCloser struct {
	io.Writer
}

func (w *nopWriteCloser) Close() error { return nil }

func NopWriteCloser(w io.Writer) io.WriteCloser {
	return &nopWriteCloser{w}
}

type bufReader struct {
	buf    *bytes.Buffer
	reader io.Reader
	err    error
	l      sync.Mutex
	wait   sync.Cond
}

func newBufReader(r io.Reader) *bufReader {
	reader := &bufReader{
		buf:    &bytes.Buffer{},
		reader: r,
	}
	reader.wait.L = &reader.l
	go reader.drain()
	return reader
}

func (r *bufReader) drain() {
	buf := make([]byte, 1024)
	for {
		n, err := r.reader.Read(buf)
		if err != nil {
			r.err = err
		} else {
			r.buf.Write(buf[0:n])
		}
		r.l.Lock()
		r.wait.Signal()
		r.l.Unlock()
		if err != nil {
			break
		}
	}
}

func (r *bufReader) Read(p []byte) (n int, err error) {
	for {
		n, err = r.buf.Read(p)
		if n > 0 {
			return n, err
		}
		if r.err != nil {
			return 0, r.err
		}
		r.l.Lock()
		r.wait.Wait()
		r.l.Unlock()
	}
	return
}

func (r *bufReader) Close() error {
	closer, ok := r.reader.(io.ReadCloser)
	if !ok {
		return nil
	}
	return closer.Close()
}

type writeBroadcaster struct {
	writers *list.List
}

func (w *writeBroadcaster) AddWriter(writer io.WriteCloser) {
	w.writers.PushBack(writer)
}

func (w *writeBroadcaster) RemoveWriter(writer io.WriteCloser) {
	for e := w.writers.Front(); e != nil; e = e.Next() {
		v := e.Value.(io.Writer)
		if v == writer {
			w.writers.Remove(e)
			return
		}
	}
}

func (w *writeBroadcaster) Write(p []byte) (n int, err error) {
	failed := []*list.Element{}
	for e := w.writers.Front(); e != nil; e = e.Next() {
		writer := e.Value.(io.Writer)
		if n, err := writer.Write(p); err != nil || n != len(p) {
			// On error, evict the writer
			failed = append(failed, e)
		}
	}
	// We cannot remove while iterating, so it has to be done in
	// a separate step
	for _, e := range failed {
		w.writers.Remove(e)
	}
	return len(p), nil
}

func (w *writeBroadcaster) Close() error {
	for e := w.writers.Front(); e != nil; e = e.Next() {
		writer := e.Value.(io.WriteCloser)
		writer.Close()
	}
	return nil
}

func newWriteBroadcaster() *writeBroadcaster {
	return &writeBroadcaster{list.New()}
}
