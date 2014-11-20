package extensions

import (
	"fmt"
	"strings"

	"github.com/docker/libpack"
)

// GitState uses docker/libpack and satisfies the State interface.
type GitState struct {
	db *libpack.DB
}

// GitStateFromFolder returns a ready-to-use GitState.
// The same folder can be used for different stores identified by storeName
// storeName can not contain slashes, as it could be mistaken for a subdirectory.
func GitStateFromFolder(folder, storeName string) (GitState, error) {
	if strings.Contains(storeName, "/") {
		return fmt.Errorf("Slashes are not allowed in storeName: %q", storeName)
	}
	db, err := libpack.OpenOrInit(folder, "refs/heads/"+storeName)
	if err != nil {
		return GitState{}, err
	}
	return GitState{db: db}, nil
}

// Close releases resources for the underlying db.
// Subsequent calls to any methods on the same GitState will result in a panic.
// To recover the same state, call GitStateFromFolder again with the same arguments.
func (s GitState) Close() {
	s.db.Free()
	s.db = nil
}

// Get returns the value associated with `key`.
func (s GitState) Get(key string) (value string, err error) {
	return s.db.Get(key)
}

// List returns a list of keys directly under dir.
// Does not walk the tree recursively.
//
// Example: /
//		foo
//			bar
//		baz
// List("/") -> ["foo", "bar"]
func (s GitState) List(dir string) ([]string, error) {
	return s.db.List(dir)
}

// Set sets the key `key`, to a value `value`.
// It automatically overrides the existing value if any.
func (s GitState) Set(key, value string) error {
	if err := s.db.Set(key, value); err != nil {
		return err
	}
	return s.db.Commit(fmt.Sprintf("set %q=%q", key, value))
}

// Remove deletes the value associated with `key`.
func (s GitState) Remove(key string) error {
	if err := s.db.Delete(key); err != nil {
		return err
	}
	return s.db.Commit(fmt.Sprintf("del %q", key))
}

// Mkdir creates the directory `dir`.
func (s GitState) Mkdir(dir string) error {
	if err := s.db.Mkdir(dir); err != nil {
		return err
	}
	return s.db.Commit(fmt.Sprintf("mkdir %q", dir))
}
