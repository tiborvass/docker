package simplebridge

import (
	"net"

	"github.com/docker/docker/pkg/iptables"
)

var chainMap = map[string]*iptables.Chain{}

func getIPTablesChain(chainName string) *iptables.Chain {
	return chainMap[chainName]
}

func forward(chainName string, action iptables.Action, proto string, sourceIP net.IP, sourcePort uint, containerIP net.IP, containerPort uint) error {
	chain := getIPTablesChain(chainName)
	return chain.Forward(action, sourceIP, sourcePort, proto, containerIP, containerPort)
}

func MapPort(chainName string, hostIP net.IP, proto string, containerIP net.IP, containerPort, hostPort uint) error {
	defer func() {
		// need to undo the iptables rules before we return
		forward(chainName, iptables.Delete, proto, hostIP, hostPort, containerIP, containerPort)
	}()

	if chainName == "" {
		chainName = "DOCKER"
	}

	if err := forward(chainName, iptables.Add, proto, hostIP, hostPort, containerIP, containerPort); err != nil {
		return err
	}

	return nil
}
