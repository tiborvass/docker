// +build daemon

package main

import (
	"log"
	"net"

	"github.com/tiborvass/docker/builtins"
	"github.com/tiborvass/docker/daemon"
	_ "github.com/tiborvass/docker/daemon/execdriver/lxc"
	_ "github.com/tiborvass/docker/daemon/execdriver/native"
	"github.com/tiborvass/docker/dockerversion"
	"github.com/tiborvass/docker/engine"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/pkg/signal"
)

const CanDaemon = true

func mainDaemon() {
	if flag.NArg() != 0 {
		flag.Usage()
		return
	}

	if *bridgeName != "" && *bridgeIp != "" {
		log.Fatal("You specified -b & --bip, mutually exclusive options. Please specify only one.")
	}

	if !*flEnableIptables && !*flInterContainerComm {
		log.Fatal("You specified --iptables=false with --icc=false. ICC uses iptables to function. Please set --icc or --iptables to true.")
	}

	if net.ParseIP(*flDefaultIp) == nil {
		log.Fatalf("Specified --ip=%s is not in correct format \"0.0.0.0\".", *flDefaultIp)
	}

	eng := engine.New()
	signal.Trap(eng.Shutdown)
	// Load builtins
	if err := builtins.Register(eng); err != nil {
		log.Fatal(err)
	}

	// load the daemon in the background so we can immediately start
	// the http api so that connections don't fail while the daemon
	// is booting
	go func() {
		// FIXME: daemon config and CLI flag parsing should be directly integrated
		cfg := &daemon.Config{
			Pidfile:                     *pidfile,
			Root:                        *flRoot,
			AutoRestart:                 *flAutoRestart,
			EnableIptables:              *flEnableIptables,
			EnableIpForward:             *flEnableIpForward,
			BridgeIP:                    *bridgeIp,
			BridgeIface:                 *bridgeName,
			DefaultIp:                   net.ParseIP(*flDefaultIp),
			InterContainerCommunication: *flInterContainerComm,
			GraphDriver:                 *flGraphDriver,
			ExecDriver:                  *flExecDriver,
			EnableSelinuxSupport:        *flSelinuxEnabled,
			GraphOptions:                flGraphOpts.GetAll(),
			Dns:                         flDns.GetAll(),
			DnsSearch:                   flDnsSearch.GetAll(),
			Mtu:                         *flMtu,
			Sockets:                     flHosts.GetAll(),
		}
		// FIXME this should be initialized in NewDaemon or somewhere in daemon.
		// Currently it is copy-pasted in `integration` to create test daemons that work.
		if cfg.Mtu == 0 {
			cfg.Mtu = daemon.GetDefaultNetworkMtu()
		}
		cfg.DisableNetwork = cfg.BridgeIface == daemon.DisableNetworkBridge

		d, err := daemon.NewDaemon(cfg, eng)
		if err != nil {
			log.Fatal(err)
		}
		if err := d.Install(eng); err != nil {
			log.Fatal(err)
		}
		// after the daemon is done setting up we can tell the api to start
		// accepting connections
		if err := eng.Job("acceptconnections").Run(); err != nil {
			log.Fatal(err)
		}
	}()
	// TODO actually have a resolved graphdriver to show?
	log.Printf("docker daemon: %s %s; execdriver: %s; graphdriver: %s",
		dockerversion.VERSION,
		dockerversion.GITCOMMIT,
		*flExecDriver,
		*flGraphDriver)

	// Serve api
	job := eng.Job("serveapi", flHosts.GetAll()...)
	job.SetenvBool("Logging", true)
	job.SetenvBool("EnableCors", *flEnableCors)
	job.Setenv("Version", dockerversion.VERSION)
	job.Setenv("SocketGroup", *flSocketGroup)

	job.SetenvBool("Tls", *flTls)
	job.SetenvBool("TlsVerify", *flTlsVerify)
	job.Setenv("TlsCa", *flCa)
	job.Setenv("TlsCert", *flCert)
	job.Setenv("TlsKey", *flKey)
	job.SetenvBool("BufferRequests", true)
	if err := job.Run(); err != nil {
		log.Fatal(err)
	}
}
