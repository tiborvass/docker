package graph

import (
	"testing"

	"github.com/tiborvass/docker/pkg/progressreader"
	"github.com/tiborvass/docker/pkg/reexec"
)

func init() {
	reexec.Init()
}

func TestPools(t *testing.T) {
	s := &TagStore{
		pullingPool: make(map[string]*progressreader.ProgressStatus),
		pushingPool: make(map[string]*progressreader.ProgressStatus),
	}

	if _, err := s.poolAdd("pull", "test1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.poolAdd("pull", "test2"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.poolAdd("push", "test1"); err == nil || err.Error() != "pull test1 is already in progress" {
		t.Fatalf("Expected `pull test1 is already in progress`")
	}
	if _, err := s.poolAdd("pull", "test1"); err == nil || err.Error() != "pull test1 is already in progress" {
		t.Fatalf("Expected `pull test1 is already in progress`")
	}
	if _, err := s.poolAdd("wait", "test3"); err == nil || err.Error() != "Unknown pool type" {
		t.Fatalf("Expected `Unknown pool type`")
	}
	if err := s.poolRemove("pull", "test2"); err != nil {
		t.Fatal(err)
	}
	if err := s.poolRemove("pull", "test2"); err != nil {
		t.Fatal(err)
	}
	if err := s.poolRemove("pull", "test1"); err != nil {
		t.Fatal(err)
	}
	if err := s.poolRemove("push", "test1"); err != nil {
		t.Fatal(err)
	}
	if err := s.poolRemove("wait", "test3"); err == nil || err.Error() != "Unknown pool type" {
		t.Fatalf("Expected `Unknown pool type`")
	}
}
