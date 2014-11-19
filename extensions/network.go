package extensions

import (
	"net"
)

// NetExtension is the interface implemented by a network extension.
// Network extensions implement new ways for Docker to interconnect containers
// over IP networks.

type NetExtension interface {
	// Opts returns a specification of options recognized by the extension.
	// The runtime uses this schema to query users for opts when loading extensionss.
	// Opts are made available to the extensions under the key "/opts" in its State.
	//
	// Schema example:
	// []Opt{
	//	{"key", "string", "A secret key to encrypt all traffic"},
	//	{"bridge", "string", "The name of the bridge interface to configure"},
	//	{"autoaddr", "bool", "Auto-detect a network range if the bridge is not already configured"},
	// }
	Opts() []Opt

	// Init initializes the extension.
	// It is called each time the  is loaded.
	Init(s State) error

	// Shutdown is called every time the extension is unloaded.
	// Note that in case of a machine crash, the extension might be loaded
	// without Shutdown having been called. It is the extension's
	// responsibility to handle this.
	Shutdown(s State) error

	// AddNet creates a new network with the name <netid>.
	// <netid> is an arbitrary string scoped to the extension.
	AddNet(netid string, s State) error

	// RemoveNet destroys the network <netid>.
	// If the network does not exist an error must be returned.
	RemoveNet(netid string, s State) error

	// AddEndpoint creates a new endpoint attached to the network <netid>.
	// <epid> is an arbitrary string, scoped to the network, which identifies the endpoint.
	// The extension must guarantee that future uses of this (netid,epid) tuple always
	// refers to the same endpoint.
	//
	// Typically an endpoint is created in 2 phases:
	//
	// 1) AddEndpoint prepares network interfaces, allocates IP addresses, and runs arbitrary
	// code specific to its backend implementation. It does this in the context of its host
	// (either the daemon process, or once out-of-process extensions are implemented, a specialized
	// container).
	//
	// 2) When AddEndpoint returns control to the core, it specifies a list of network interfaces
	// to use for this endpoint, along with the configuration to apply. The core then finalizes
	// the configuration on behalf of the extension.
	//
	// Note: the reason the core is responsible for configuration, and not the extension, is that
	// the core may move the interfaces to a different network namespace: in that case
	// the configuration must be applied *after* the interface is moved. Since the core takes
	// care of that, the extensions don't need to know anything about namespace manipulation, which
	// helps keep them simple and portable.
	AddEndpoint(netid, epid string, s State) ([]*Interface, error)

	// RemoveEndpoint
	RemoveEndpoint(netid, epid string, s State) error
}

type Opt struct {
	Key  string
	Type string
	Desc string
}

// this is a network configuration provided by the extension at network creation
// time. it is not intended to be malleable by docker itself, but should be
// populated with all the generic configuration required by dockerinit to work
// with the container from a network perspective.
type Interface struct {
	net.Interface
	Addresses []string // a list of CIDR addresses to assign to the interface
	Gateway   net.IP   // the gateway the interface uses
	Routes    []Route  // routes to set on this interface
}

type Route struct {
	Target  string // The CIDR address of the route target
	Gateway string // The CIDR address of the gateway (empty string means no gateway)
}

// State is an abstract database provided by the core for a extension to easily store
// and retrieve its state across its lifecycle.
//
// State data is organized like a simplified filesystem: nodes in the tree
// are referenced by slash-separated paths. A node can be either a directory
// or a file. Files have no metadata, only data.
//
// An important property of State is full support of versioning and transactions.
// All changes return a globally unique hash of the current state of the database.
//
// State supports transactions:
// [...]
// var (
//    s State
//    err error
//  )
// s.Autocommit(false)
// s, _ = s.("/foo/bar")
// s, _ = s.Set("/animals/moby dock", "Moby Dock is a whale")
// _ = s.Commit()

type State interface {
	List(dir string) ([]string, error)
	Get(key string) (string, error)

	// FIXME: for now we stick to naive crud,
	// bring back transactions and versioning once
	// we have a working poc.
	Set(key, val string) error
	Remove(key string) error
	Mkdir(dir string) error
}
