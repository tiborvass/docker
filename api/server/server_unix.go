// +build freebsd linux

package server

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/tiborvass/docker/pkg/sockets"
	"github.com/docker/libnetwork/portallocator"

	systemdActivation "github.com/coreos/go-systemd/activation"
)

// newServer sets up the required HTTPServers and does protocol specific checking.
// newServer does not set any muxers, you should set it later to Handler field
func (s *Server) newServer(proto, addr string) ([]*HTTPServer, error) {
	var (
		err error
		ls  []net.Listener
	)
	switch proto {
	case "fd":
		ls, err = listenFD(addr)
		if err != nil {
			return nil, err
		}
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
	var res []*HTTPServer
	for _, l := range ls {
		res = append(res, &HTTPServer{
			&http.Server{
				Addr: addr,
			},
			l,
		})
	}
	return res, nil
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

// listenFD returns the specified socket activated files as a slice of
// net.Listeners or all of the activated files if "*" is given.
func listenFD(addr string) ([]net.Listener, error) {
	// socket activation
	listeners, err := systemdActivation.Listeners(false)
	if err != nil {
		return nil, err
	}

	if listeners == nil || len(listeners) == 0 {
		return nil, fmt.Errorf("No sockets found")
	}

	// default to all fds just like unix:// and tcp://
	if addr == "" {
		addr = "*"
	}

	fdNum, _ := strconv.Atoi(addr)
	fdOffset := fdNum - 3
	if (addr != "*") && (len(listeners) < int(fdOffset)+1) {
		return nil, fmt.Errorf("Too few socket activated files passed in")
	}

	if addr == "*" {
		return listeners, nil
	}

	return []net.Listener{listeners[fdOffset]}, nil
}
