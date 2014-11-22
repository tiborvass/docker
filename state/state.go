package state

import (
	"io"

	"github.com/docker/docker/core"
)

// FIXME:networking Just to get things to build
type State interface {
	Scope(core.DID) State
}

// github.com/docker/docker/state
// A minimalist database for managing and distributing Docker's state
//
// LOCK tibor and shykes

// Value is either a Tree or a string
type Value interface{}

func NewDBFromDriver(name string, d StateDriver) (DB, error) {
	return nil, nil
}

type DB interface {
	Get() Tree
	Watch(key string) (Value, chan Value)
	Set(val Tree)
}

type Tree interface {
	Get(key string) (string, error)
	Set(key, val string) (Tree, error)
	Mkdir(key string) (Tree, error)
	Remove(key string) (Tree, error)
	Diff(other Tree) (added, removed Tree)
	Walk(func(key string, entry Value)) error
	Add(key string, overlay Tree) (Tree, error)
	Substract(key string, whiteout Tree) (Tree, error)
	Scope(key string) (Tree, error)
	Pipeline() Pipeline
}

type Pipeline interface {
	Run() (Tree, error)

	Set(key, val string) Pipeline
	Mkdir(key string) Pipeline
	Remove(key string) Pipeline
	Add(key string, overlay Tree) Pipeline
	Substract(key string, whiteout Tree) Pipeline
	Scope(key string) Pipeline
}

type StateDriver interface {
	// These 4 methods map to the methods of a native libgit2 obj db backend
	ListObjects() ([]string, error)
	ReadObject(key_or_prefix string) (io.Reader, error)
	AddObject(key string, value io.Reader) error
	DeleteObject(key string) error

	// These 5 methods map to the methods of a native libgit2 ref db backend
	ListRefs() ([]string, error)
	GetRef(key string) (string, error)
	SetRef(key, value string) error
	RenameRef(old, new string) error
	DeleteRef(key string) error
}
