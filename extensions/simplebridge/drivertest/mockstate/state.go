package mockstate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/docker/docker/state"
)

type State struct {
	State map[string]string
	mutex sync.Mutex
}

func (s *State) Add(key string, overlay state.Tree) (state.Tree, error)       { return nil, nil }
func (s *State) Walk(func(key string, entry state.Value)) error               { return nil }
func (s *State) Mkdir(key string) (state.Tree, error)                         { return nil, nil }
func (s *State) Diff(other state.Tree) (added, removed state.Tree)            { return nil, nil }
func (s *State) Subtract(key string, whiteout state.Tree) (state.Tree, error) { return nil, nil }
func (s *State) Pipeline() state.Pipeline                                     { return nil }

func (s *State) Remove(key string) (state.Tree, error) {
	for thiskey := range s.State {
		if ok, _ := path.Match(path.Join(key, "*"), thiskey); ok {
			delete(s.State, thiskey)
		}
	}
	return s, nil
}

func (s *State) List(dir string) ([]string, error) {
	result := []string{}

	for key := range s.State {
		fmt.Println(dir, " ", key)
		thisdir := path.Dir(key)

		dirok, _ := path.Match(path.Join(dir, "*"), thisdir)
		fileok, _ := path.Match(path.Join(dir, "*"), key)

		if !dirok && !fileok {
			fmt.Println("skipping")
			continue
		} else if !dirok {
			fmt.Println("appending", key)
			result = append(result, path.Base(key))
		} else {
			fmt.Println("appending", thisdir)
			result = append(result, path.Base(thisdir))
		}
	}

	return result, nil
}

func (s *State) Scope(id string) (state.Tree, error) {
	return s, nil
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
	fmt.Println(id, " ", s.State[id])
	return s.State[id], nil
}

func (s *State) Set(key, val string) (state.Tree, error) {
	s.mutex.Lock()
	s.State[key] = val
	s.mutex.Unlock()
	return s, nil
}
