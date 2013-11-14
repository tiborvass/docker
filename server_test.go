package docker

import (
	"github.com/dotcloud/docker/utils"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestContainerTagImageDelete(t *testing.T) {
	runtime := mkRuntime(t)
	defer nuke(runtime)

	srv := &Server{runtime: runtime}

	initialImages, err := srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := srv.runtime.repositories.Set("utest", "tag1", unitTestImageName, false); err != nil {
		t.Fatal(err)
	}

	if err := srv.runtime.repositories.Set("utest/docker", "tag2", unitTestImageName, false); err != nil {
		t.Fatal(err)
	}
	if err := srv.runtime.repositories.Set("utest:5000/docker", "tag3", unitTestImageName, false); err != nil {
		t.Fatal(err)
	}

	images, err := srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(images[0].RepoTags) != len(initialImages[0].RepoTags)+3 {
		t.Errorf("Expected %d images, %d found", len(initialImages)+3, len(images))
	}

	if _, err := srv.ImageDelete("utest/docker:tag2", true); err != nil {
		t.Fatal(err)
	}

	images, err = srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(images[0].RepoTags) != len(initialImages[0].RepoTags)+2 {
		t.Errorf("Expected %d images, %d found", len(initialImages)+2, len(images))
	}

	if _, err := srv.ImageDelete("utest:5000/docker:tag3", true); err != nil {
		t.Fatal(err)
	}

	images, err = srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(images[0].RepoTags) != len(initialImages[0].RepoTags)+1 {
		t.Errorf("Expected %d images, %d found", len(initialImages)+1, len(images))
	}

	if _, err := srv.ImageDelete("utest:tag1", true); err != nil {
		t.Fatal(err)
	}

	images, err = srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != len(initialImages) {
		t.Errorf("Expected %d image, %d found", len(initialImages), len(images))
	}
}

func TestCreateRm(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	runtime := srv.runtime
	defer nuke(runtime)

	config, _, _, err := ParseRun([]string{GetTestImage(runtime).ID, "echo test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	if len(runtime.List()) != 1 {
		t.Errorf("Expected 1 container, %v found", len(runtime.List()))
	}

	if err = srv.ContainerDestroy(id, true, false); err != nil {
		t.Fatal(err)
	}

	if len(runtime.List()) != 0 {
		t.Errorf("Expected 0 container, %v found", len(runtime.List()))
	}

}

func TestCreateRmVolumes(t *testing.T) {
	eng := NewTestEngine(t)

	srv := mkServerFromEngine(eng, t)
	runtime := srv.runtime
	defer nuke(runtime)

	config, hostConfig, _, err := ParseRun([]string{"-v", "/srv", GetTestImage(runtime).ID, "echo test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	if len(runtime.List()) != 1 {
		t.Errorf("Expected 1 container, %v found", len(runtime.List()))
	}

	job := eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	err = srv.ContainerStop(id, 1)
	if err != nil {
		t.Fatal(err)
	}

	if err = srv.ContainerDestroy(id, true, false); err != nil {
		t.Fatal(err)
	}

	if len(runtime.List()) != 0 {
		t.Errorf("Expected 0 container, %v found", len(runtime.List()))
	}
}

func TestCommit(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	runtime := srv.runtime
	defer nuke(runtime)

	config, _, _, err := ParseRun([]string{GetTestImage(runtime).ID, "/bin/cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	if _, err := srv.ContainerCommit(id, "testrepo", "testtag", "", "", config); err != nil {
		t.Fatal(err)
	}
}

func TestCreateStartRestartStopStartKillRm(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	runtime := srv.runtime
	defer nuke(runtime)

	config, hostConfig, _, err := ParseRun([]string{GetTestImage(runtime).ID, "/bin/cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	if len(runtime.List()) != 1 {
		t.Errorf("Expected 1 container, %v found", len(runtime.List()))
	}

	job := eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if err := srv.ContainerRestart(id, 15); err != nil {
		t.Fatal(err)
	}

	if err := srv.ContainerStop(id, 15); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if err := srv.ContainerKill(id, 0); err != nil {
		t.Fatal(err)
	}

	// FIXME: this failed once with a race condition ("Unable to remove filesystem for xxx: directory not empty")
	if err := srv.ContainerDestroy(id, true, false); err != nil {
		t.Fatal(err)
	}

	if len(runtime.List()) != 0 {
		t.Errorf("Expected 0 container, %v found", len(runtime.List()))
	}

}

func TestRunWithTooLowMemoryLimit(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	runtime := srv.runtime
	defer nuke(runtime)

	// Try to create a container with a memory limit of 1 byte less than the minimum allowed limit.
	job := eng.Job("create")
	job.Setenv("Image", GetTestImage(runtime).ID)
	job.Setenv("Memory", "524287")
	job.Setenv("CpuShares", "1000")
	job.SetenvList("Cmd", []string{"/bin/cat"})
	var id string
	job.StdoutParseString(&id)
	if err := job.Run(); err == nil {
		t.Errorf("Memory limit is smaller than the allowed limit. Container creation should've failed!")
	}
}

func TestContainerTop(t *testing.T) {
	t.Skip("Fixme. Skipping test for now. Reported error: 'server_test.go:236: Expected 2 processes, found 1.'")

	runtime := mkRuntime(t)
	defer nuke(runtime)

	srv := &Server{runtime: runtime}

	c, _ := mkContainer(runtime, []string{"_", "/bin/sh", "-c", "sleep 2"}, t)
	c, err := mkContainer(runtime, []string{"_", "/bin/sh", "-c", "sleep 2"}, t)
	if err != nil {
		t.Fatal(err)
	}

	defer runtime.Destroy(c)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}

	// Give some time to the process to start
	c.WaitTimeout(500 * time.Millisecond)

	if !c.State.Running {
		t.Errorf("Container should be running")
	}
	procs, err := srv.ContainerTop(c.ID, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(procs.Processes) != 2 {
		t.Fatalf("Expected 2 processes, found %d.", len(procs.Processes))
	}

	pos := -1
	for i := 0; i < len(procs.Titles); i++ {
		if procs.Titles[i] == "CMD" {
			pos = i
			break
		}
	}

	if pos == -1 {
		t.Fatalf("Expected CMD, not found.")
	}

	if procs.Processes[0][pos] != "sh" && procs.Processes[0][pos] != "busybox" {
		t.Fatalf("Expected `busybox` or `sh`, found %s.", procs.Processes[0][pos])
	}

	if procs.Processes[1][pos] != "sh" && procs.Processes[1][pos] != "busybox" {
		t.Fatalf("Expected `busybox` or `sh`, found %s.", procs.Processes[1][pos])
	}
}

func TestPools(t *testing.T) {
	runtime := mkRuntime(t)
	srv := &Server{
		runtime:     runtime,
		pullingPool: make(map[string]struct{}),
		pushingPool: make(map[string]struct{}),
	}
	defer nuke(runtime)

	err := srv.poolAdd("pull", "test1")
	if err != nil {
		t.Fatal(err)
	}
	err = srv.poolAdd("pull", "test2")
	if err != nil {
		t.Fatal(err)
	}
	err = srv.poolAdd("push", "test1")
	if err == nil || err.Error() != "pull test1 is already in progress" {
		t.Fatalf("Expected `pull test1 is already in progress`")
	}
	err = srv.poolAdd("pull", "test1")
	if err == nil || err.Error() != "pull test1 is already in progress" {
		t.Fatalf("Expected `pull test1 is already in progress`")
	}
	err = srv.poolAdd("wait", "test3")
	if err == nil || err.Error() != "Unknown pool type" {
		t.Fatalf("Expected `Unknown pool type`")
	}

	err = srv.poolRemove("pull", "test2")
	if err != nil {
		t.Fatal(err)
	}
	err = srv.poolRemove("pull", "test2")
	if err != nil {
		t.Fatal(err)
	}
	err = srv.poolRemove("pull", "test1")
	if err != nil {
		t.Fatal(err)
	}
	err = srv.poolRemove("push", "test1")
	if err != nil {
		t.Fatal(err)
	}
	err = srv.poolRemove("wait", "test3")
	if err == nil || err.Error() != "Unknown pool type" {
		t.Fatalf("Expected `Unknown pool type`")
	}
}

func TestLogEvent(t *testing.T) {
	runtime := mkRuntime(t)
	defer nuke(runtime)
	srv := &Server{
		runtime:   runtime,
		events:    make([]utils.JSONMessage, 0, 64),
		listeners: make(map[string]chan utils.JSONMessage),
	}

	srv.LogEvent("fakeaction", "fakeid", "fakeimage")

	listener := make(chan utils.JSONMessage)
	srv.Lock()
	srv.listeners["test"] = listener
	srv.Unlock()

	srv.LogEvent("fakeaction2", "fakeid", "fakeimage")

	if len(srv.events) != 2 {
		t.Fatalf("Expected 2 events, found %d", len(srv.events))
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		srv.LogEvent("fakeaction3", "fakeid", "fakeimage")
		time.Sleep(200 * time.Millisecond)
		srv.LogEvent("fakeaction4", "fakeid", "fakeimage")
	}()

	setTimeout(t, "Listening for events timed out", 2*time.Second, func() {
		for i := 2; i < 4; i++ {
			event := <-listener
			if event != srv.events[i] {
				t.Fatalf("Event received it different than expected")
			}
		}
	})
}

func TestRmi(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	runtime := srv.runtime
	defer nuke(runtime)

	initialImages, err := srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	config, hostConfig, _, err := ParseRun([]string{GetTestImage(runtime).ID, "echo test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	containerID := createTestContainer(eng, config, t)

	//To remove
	job := eng.Job("start", containerID)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	imageID, err := srv.ContainerCommit(containerID, "test", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = srv.ContainerTag(imageID, "test", "0.1", false)
	if err != nil {
		t.Fatal(err)
	}

	containerID = createTestContainer(eng, config, t)

	//To remove
	job = eng.Job("start", containerID)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	_, err = srv.ContainerCommit(containerID, "test", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	images, err := srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(images)-len(initialImages) != 2 {
		t.Fatalf("Expected 2 new images, found %d.", len(images)-len(initialImages))
	}

	_, err = srv.ImageDelete(imageID, true)
	if err != nil {
		t.Fatal(err)
	}

	images, err = srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(images)-len(initialImages) != 1 {
		t.Fatalf("Expected 1 new image, found %d.", len(images)-len(initialImages))
	}

	for _, image := range images {
		if strings.Contains(unitTestImageID, image.ID) {
			continue
		}
		if image.RepoTags[0] == "<none>:<none>" {
			t.Fatalf("Expected tagged image, got untagged one.")
		}
	}
}

func TestImagesFilter(t *testing.T) {
	runtime := mkRuntime(t)
	defer nuke(runtime)

	srv := &Server{runtime: runtime}

	if err := srv.runtime.repositories.Set("utest", "tag1", unitTestImageName, false); err != nil {
		t.Fatal(err)
	}

	if err := srv.runtime.repositories.Set("utest/docker", "tag2", unitTestImageName, false); err != nil {
		t.Fatal(err)
	}
	if err := srv.runtime.repositories.Set("utest:5000/docker", "tag3", unitTestImageName, false); err != nil {
		t.Fatal(err)
	}

	images, err := srv.Images(false, "utest*/*")
	if err != nil {
		t.Fatal(err)
	}

	if len(images[0].RepoTags) != 2 {
		t.Fatal("incorrect number of matches returned")
	}

	images, err = srv.Images(false, "utest")
	if err != nil {
		t.Fatal(err)
	}

	if len(images[0].RepoTags) != 1 {
		t.Fatal("incorrect number of matches returned")
	}

	images, err = srv.Images(false, "utest*")
	if err != nil {
		t.Fatal(err)
	}

	if len(images[0].RepoTags) != 1 {
		t.Fatal("incorrect number of matches returned")
	}

	images, err = srv.Images(false, "*5000*/*")
	if err != nil {
		t.Fatal(err)
	}

	if len(images[0].RepoTags) != 1 {
		t.Fatal("incorrect number of matches returned")
	}
}

func TestImageInsert(t *testing.T) {
	runtime := mkRuntime(t)
	defer nuke(runtime)
	srv := &Server{runtime: runtime}
	sf := utils.NewStreamFormatter(true)

	// bad image name fails
	if err := srv.ImageInsert("foo", "https://www.docker.io/static/img/docker-top-logo.png", "/foo", ioutil.Discard, sf); err == nil {
		t.Fatal("expected an error and got none")
	}

	// bad url fails
	if err := srv.ImageInsert(GetTestImage(runtime).ID, "http://bad_host_name_that_will_totally_fail.com/", "/foo", ioutil.Discard, sf); err == nil {
		t.Fatal("expected an error and got none")
	}

	// success returns nil
	if err := srv.ImageInsert(GetTestImage(runtime).ID, "https://www.docker.io/static/img/docker-top-logo.png", "/foo", ioutil.Discard, sf); err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}
}
