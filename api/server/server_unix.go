// +build freebsd linux

package server

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/pkg/sockets"
	"github.com/tiborvass/docker/pkg/systemd"
	"github.com/docker/libnetwork/portallocator"
)

// newServer sets up the required serverClosers and does protocol specific checking.
func (s *Server) newServer(proto, addr string) ([]serverCloser, error) {
	var (
		err error
		ls  []net.Listener
	)
	switch proto {
	case "fd":
		ls, err = systemd.ListenFD(addr)
		if err != nil {
			return nil, err
		}
		// We don't want to start serving on these sockets until the
		// daemon is initialized and installed. Otherwise required handlers
		// won't be ready.
		<-s.start
	case "tcp":
		l, err := s.initTCPSocket(addr)
		if err != nil {
			return nil, err
		}
		ls = append(ls, l)
	case "unix":
		l, err := sockets.NewUnixSocket(addr, s.cfg.SocketGroup, s.start)
		if err != nil {
			return nil, err
		}
		ls = append(ls, l)
	default:
		return nil, fmt.Errorf("Invalid protocol format: %q", proto)
	}
	var res []serverCloser
	for _, l := range ls {
		res = append(res, &HTTPServer{
			&http.Server{
				Addr:    addr,
				Handler: s.router,
			},
			l,
		})
	}
	return res, nil
}

// AcceptConnections allows clients to connect to the API server.
// Referenced Daemon is notified about this server, and waits for the
// daemon acknowledgement before the incoming connections are accepted.
func (s *Server) AcceptConnections(d *daemon.Daemon) {
	// Tell the init daemon we are accepting requests
	s.daemon = d
	s.registerSubRouter()
	go systemd.SdNotify("READY=1")
	// close the lock so the listeners start accepting connections
	select {
	case <-s.start:
	default:
		close(s.start)
	}
}

func allocateDaemonPort(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	intPort, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	var hostIPs []net.IP
	if parsedIP := net.ParseIP(host); parsedIP != nil {
		hostIPs = append(hostIPs, parsedIP)
	} else if hostIPs, err = net.LookupIP(host); err != nil {
		return fmt.Errorf("failed to lookup %s address in host specification", host)
	}

	pa := portallocator.Get()
	for _, hostIP := range hostIPs {
		if _, err := pa.RequestPort(hostIP, "tcp", intPort); err != nil {
			return fmt.Errorf("failed to allocate daemon listening port %d (err: %v)", intPort, err)
		}
	}
	return nil
}

// getContainersByNameDownlevel performs processing for pre 1.20 APIs. This
// is only relevant on non-Windows daemons.
func getContainersByNameDownlevel(w http.ResponseWriter, s *Server, namevar string) error {
	containerJSONRaw, err := s.daemon.ContainerInspectPre120(namevar)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, containerJSONRaw)
}
