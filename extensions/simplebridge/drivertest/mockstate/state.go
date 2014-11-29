package mockstate

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"

	"github.com/docker/docker/core"
	"github.com/docker/docker/state"
)

type State struct {
	State map[string]string
	mutex sync.Mutex
}

type Tree struct{}

func (s *State) Scope(id core.DID) state.State {
	return s
}

func (s *State) Save() error {
	f, err := os.Create("state.json")
	if err != nil {
		return err
	}

	content, err := json.Marshal(s)
	if err != nil {
		return err
	}

	if _, err := f.Write(content); err != nil {
		return err
	}

	return nil
}

func NewState() *State {
	return &State{
		State: map[string]string{},
	}
}

func Load() *State {
	f, err := os.Open("state.json")
	if err != nil {
		return NewState()
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return NewState()
	}

	var state State

	if err := json.Unmarshal(content, &state); err != nil {
		return NewState()
	}

	return &state
}

func (s *State) Get(id string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.State[id], nil
}

func (s *State) Set(key, val string) (Tree, error) {
	s.mutex.Lock()
	s.State[key] = val
	s.mutex.Unlock()
	return Tree{}, nil
}
