package net

import (
	"fmt"

	"github.com/docker/docker/core"
	"github.com/docker/docker/engine"
	"github.com/docker/docker/network"
	"github.com/docker/docker/sandbox"
)

func New(netController *network.Controller, sandboxController sandbox.Controller) *Service {
	return &Service{
		netController:     netController,
		sandboxController: sandboxController,
	}
}

type Service struct {
	// Containers at creation time will create an Endpoint on the default
	// network identified by this ID.
	DefaultNetworkID core.DID

	// Controllers provide access to the collection of networks, and the
	// collection of sandboxes. Joining a network is drawing a line between a
	// given sandbox and a given sandbox.
	netController     *network.Controller
	sandboxController sandbox.Controller
}

func (s *Service) Install(eng *engine.Engine) error {
	for name, handler := range map[string]engine.Handler{
		"net_create": s.CmdCreate,
		"net_export": s.CmdExport,
		"net_import": s.CmdImport,
		"net_join":   s.CmdJoin,
		"net_ls":     s.CmdLs,
		"net_leave":  s.CmdLeave,
		"net_rm":     s.CmdRm,
	} {
		if err := eng.Register(name, handler); err != nil {
			return fmt.Errorf("failed to register %q: %v\n", name, err)
		}
	}
	return nil
}

func (s *Service) CmdCreate(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("usage: %s NAME", job.Name)
	}

	// FIXME What do we do with user provided name?
	// Store in Service? Store in NetController?
	netw, err := s.netController.NewNetwork()
	if err != nil {
		return job.Error(err)
	}
	job.Printf("%v\n", netw.Id())
	return engine.StatusOK
}

func (s *Service) CmdLs(job *engine.Job) engine.Status {
	netw := s.netController.ListNetworks()
	table := engine.NewTable("Name", len(netw))
	for _, netid := range netw {
		item := &engine.Env{}
		item.Set("ID", string(netid))
	}

	if _, err := table.WriteTo(job.Stdout); err != nil {
		return job.Error(err)
	}
	return engine.StatusOK
}

func (s *Service) CmdRm(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("usage: %s NAME", job.Name)
	}

	if err := s.netController.RemoveNetwork(core.DID(job.Args[0])); err != nil {
		return job.Error(err)
	}
	return engine.StatusOK
}

func (s *Service) CmdJoin(job *engine.Job) engine.Status {
	if len(job.Args) != 3 {
		return job.Errorf("usage: %s NETWORK CONTAINER NAME", job.Name)
	}

	net, err := s.netController.GetNetwork(core.DID(job.Args[0]))
	if err != nil {
		return job.Error(err)
	}

	// FIXME The provided CONTAINER could be the 'user facing ID'. but not
	// necessarily the sandbox ID itself: we're keeping things simple herengine.
	sandbox, err := s.sandboxController.Get(core.DID(job.Args[1]))
	if err != nil {
		return job.Error(err)
	}

	if _, err := net.Link(sandbox, job.Args[2], false); err != nil {
		return job.Error(err)
	}
	return engine.StatusOK
}

func (s *Service) CmdLeave(job *engine.Job) engine.Status {
	if len(job.Args) != 2 {
		return job.Errorf("usage: %s NETWORK NAME", job.Name)
	}

	net, err := s.netController.GetNetwork(core.DID(job.Args[0]))
	if err != nil {
		return job.Error(err)
	}

	if err := net.Unlink(job.Args[1]); err != nil {
		return job.Error(err)
	}
	return engine.StatusOK
}

func (s *Service) CmdImport(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("usage: %s NAME", job.Name)
	}
	// FIXME
	return engine.StatusOK
}

func (s *Service) CmdExport(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("usage: %s NAME", job.Name)
	}
	// FIXME
	return engine.StatusOK
}
