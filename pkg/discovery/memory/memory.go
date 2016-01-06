package memory

import (
	"time"

	"github.com/tiborvass/docker/pkg/discovery"
)

// Discovery implements a descovery backend that keeps
// data in memory.
type Discovery struct {
	heartbeat time.Duration
	values    []string
}

func init() {
	Init()
}

// Init registers the memory backend on demand.
func Init() {
	discovery.Register("memory", &Discovery{})
}

// Initialize sets the heartbeat for the memory backend.
func (s *Discovery) Initialize(_ string, heartbeat time.Duration, _ time.Duration, _ map[string]string) error {
	s.heartbeat = heartbeat
	return nil
}

// Watch sends periodic discovery updates to a channel.
func (s *Discovery) Watch(stopCh <-chan struct{}) (<-chan discovery.Entries, <-chan error) {
	ch := make(chan discovery.Entries)
	errCh := make(chan error)
	ticker := time.NewTicker(s.heartbeat)

	go func() {
		defer close(errCh)
		defer close(ch)

		// Send the initial entries if available.
		var currentEntries discovery.Entries
		if len(s.values) > 0 {
			var err error
			currentEntries, err = discovery.CreateEntries(s.values)
			if err != nil {
				errCh <- err
			} else {
				ch <- currentEntries
			}
		}

		// Periodically send updates.
		for {
			select {
			case <-ticker.C:
				newEntries, err := discovery.CreateEntries(s.values)
				if err != nil {
					errCh <- err
					continue
				}

				// Check if the file has really changed.
				if !newEntries.Equals(currentEntries) {
					ch <- newEntries
				}
				currentEntries = newEntries
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	return ch, errCh
}

// Register adds a new address to the discovery.
func (s *Discovery) Register(addr string) error {
	s.values = append(s.values, addr)
	return nil
}
