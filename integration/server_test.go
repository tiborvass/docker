package docker

import (
	"github.com/dotcloud/docker"
	"github.com/dotcloud/docker/utils"
	"io/ioutil"
	"strings"
	"testing"
)

func TestImageTagImageDelete(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	srv := mkServerFromEngine(eng, t)

	initialImages, err := srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.Job("tag", unitTestImageName, "utest", "tag1").Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", unitTestImageName, "utest/docker", "tag2").Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", unitTestImageName, "utest:5000/docker", "tag3").Run(); err != nil {
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
	defer mkRuntimeFromEngine(eng, t).Nuke()

	config, _, _, err := docker.ParseRun([]string{unitTestImageID, "echo test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	if c := srv.Containers(true, false, -1, "", ""); len(c) != 1 {
		t.Errorf("Expected 1 container, %v found", len(c))
	}

	job := eng.Job("container_delete", id)
	job.SetenvBool("removeVolume", true)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if c := srv.Containers(true, false, -1, "", ""); len(c) != 0 {
		t.Errorf("Expected 0 container, %v found", len(c))
	}

}

func TestCreateRmVolumes(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	config, hostConfig, _, err := docker.ParseRun([]string{"-v", "/srv", unitTestImageID, "echo", "test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	if c := srv.Containers(true, false, -1, "", ""); len(c) != 1 {
		t.Errorf("Expected 1 container, %v found", len(c))
	}

	job := eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("stop", id)
	job.SetenvInt("t", 1)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("container_delete", id)
	job.SetenvBool("removeVolume", true)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if c := srv.Containers(true, false, -1, "", ""); len(c) != 0 {
		t.Errorf("Expected 0 container, %v found", len(c))
	}
}

func TestCommit(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	config, _, _, err := docker.ParseRun([]string{unitTestImageID, "/bin/cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	job := eng.Job("commit", id)
	job.Setenv("repo", "testrepo")
	job.Setenv("tag", "testtag")
	job.SetenvJson("config", config)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateStartRestartStopStartKillRm(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	config, hostConfig, _, err := docker.ParseRun([]string{"-i", unitTestImageID, "/bin/cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	if c := srv.Containers(true, false, -1, "", ""); len(c) != 1 {
		t.Errorf("Expected 1 container, %v found", len(c))
	}

	job := eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("restart", id)
	job.SetenvInt("t", 15)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("stop", id)
	job.SetenvInt("t", 15)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("kill", id).Run(); err != nil {
		t.Fatal(err)
	}

	// FIXME: this failed once with a race condition ("Unable to remove filesystem for xxx: directory not empty")
	job = eng.Job("container_delete", id)
	job.SetenvBool("removeVolume", true)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if c := srv.Containers(true, false, -1, "", ""); len(c) != 0 {
		t.Errorf("Expected 0 container, %v found", len(c))
	}
}

func TestRunWithTooLowMemoryLimit(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	// Try to create a container with a memory limit of 1 byte less than the minimum allowed limit.
	job := eng.Job("create")
	job.Setenv("Image", unitTestImageID)
	job.Setenv("Memory", "524287")
	job.Setenv("CpuShares", "1000")
	job.SetenvList("Cmd", []string{"/bin/cat"})
	var id string
	job.Stdout.AddString(&id)
	if err := job.Run(); err == nil {
		t.Errorf("Memory limit is smaller than the allowed limit. Container creation should've failed!")
	}
}

func TestRmi(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	initialImages, err := srv.Images(false, "")
	if err != nil {
		t.Fatal(err)
	}

	config, hostConfig, _, err := docker.ParseRun([]string{unitTestImageID, "echo", "test"}, nil)
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

	if err := eng.Job("wait", containerID).Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("commit", containerID)
	job.Setenv("repo", "test")
	var imageID string
	job.Stdout.AddString(&imageID)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", imageID, "test", "0.1").Run(); err != nil {
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

	if err := eng.Job("wait", containerID).Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("commit", containerID)
	job.Setenv("repo", "test")
	if err := job.Run(); err != nil {
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
	eng := NewTestEngine(t)
	defer nuke(mkRuntimeFromEngine(eng, t))

	srv := mkServerFromEngine(eng, t)

	if err := eng.Job("tag", unitTestImageName, "utest", "tag1").Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", unitTestImageName, "utest/docker", "tag2").Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", unitTestImageName, "utest:5000/docker", "tag3").Run(); err != nil {
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
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()
	srv := mkServerFromEngine(eng, t)
	sf := utils.NewStreamFormatter(true)

	// bad image name fails
	if err := srv.ImageInsert("foo", "https://www.docker.io/static/img/docker-top-logo.png", "/foo", ioutil.Discard, sf); err == nil {
		t.Fatal("expected an error and got none")
	}

	// bad url fails
	if err := srv.ImageInsert(unitTestImageID, "http://bad_host_name_that_will_totally_fail.com/", "/foo", ioutil.Discard, sf); err == nil {
		t.Fatal("expected an error and got none")
	}

	// success returns nil
	if err := srv.ImageInsert(unitTestImageID, "https://www.docker.io/static/img/docker-top-logo.png", "/foo", ioutil.Discard, sf); err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}
}

// Regression test for being able to untag an image with an existing
// container
func TestDeleteTagWithExistingContainers(t *testing.T) {
	eng := NewTestEngine(t)
	defer nuke(mkRuntimeFromEngine(eng, t))

	srv := mkServerFromEngine(eng, t)

	// Tag the image
	if err := eng.Job("tag", unitTestImageID, "utest", "tag1").Run(); err != nil {
		t.Fatal(err)
	}

	// Create a container from the image
	config, _, _, err := docker.ParseRun([]string{unitTestImageID, "echo test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createNamedTestContainer(eng, config, t, "testingtags")
	if id == "" {
		t.Fatal("No id returned")
	}

	containers := srv.Containers(true, false, -1, "", "")

	if len(containers) != 1 {
		t.Fatalf("Expected 1 container got %d", len(containers))
	}

	// Try to remove the tag
	imgs, err := srv.ImageDelete("utest:tag1", true)
	if err != nil {
		t.Fatal(err)
	}

	if len(imgs) != 1 {
		t.Fatalf("Should only have deleted one untag %d", len(imgs))
	}

	untag := imgs[0]

	if untag.Untagged != unitTestImageID {
		t.Fatalf("Expected %s got %s", unitTestImageID, untag.Untagged)
	}
}
