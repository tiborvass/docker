package docker

import (
	"testing"
)

func TestServerListOrderedImages(t *testing.T) {
	runtime := mkRuntime(t)
	defer nuke(runtime)

	archive, err := fakeTar()
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.graph.Create(archive, nil, "Testing", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	srv := &Server{runtime: runtime}

	images, err := srv.Images(true, "")
	if err != nil {
		t.Fatal(err)
	}

	if images[0].Created < images[1].Created {
		t.Error("Expected []APIImges to be ordered by most recent creation date.")
	}
}
