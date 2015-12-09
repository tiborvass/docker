package network

import (
	"github.com/tiborvass/docker/api/types/network"

	"github.com/docker/libnetwork"
)

// Backend is all the methods that need to be implemented to provide
// network specific functionality
type Backend interface {
	FindNetwork(idName string) (libnetwork.Network, error)
	GetNetwork(idName string, by int) (libnetwork.Network, error)
	GetNetworksByID(partialID string) []libnetwork.Network
	CreateNetwork(name, driver string, ipam network.IPAM,
		options map[string]string) (libnetwork.Network, error)
	ConnectContainerToNetwork(containerName, networkName string) error
	DisconnectContainerFromNetwork(containerName string,
		network libnetwork.Network) error
	NetworkControllerEnabled() bool
}
