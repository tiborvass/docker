package main

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/tiborvass/docker/api/types"
)

func TestLegacyImages(t *testing.T) {
	_, body, err := sockRequest("GET", "/v1.6/images/json", nil)
	if err != nil {
		t.Fatalf("Error on GET: %s", err)
	}

	images := []types.LegacyImage{}
	if err = json.Unmarshal(body, &images); err != nil {
		t.Fatalf("Error on unmarshal: %s", err)
	}

	if len(images) == 0 || images[0].Tag == "" || images[0].Repository == "" {
		t.Fatalf("Bad data: %q", images)
	}

	logDone("images - checking legacy json")
}

func TestApiImagesFilter(t *testing.T) {
	name := "utest:tag1"
	name2 := "utest/docker:tag2"
	name3 := "utest:5000/docker:tag3"
	defer deleteImages(name, name2, name3)
	dockerCmd(t, "tag", "busybox", name)
	dockerCmd(t, "tag", "busybox", name2)
	dockerCmd(t, "tag", "busybox", name3)

	type image struct{ RepoTags []string }
	getImages := func(filter string) []image {
		v := url.Values{}
		v.Set("filter", filter)
		_, b, err := sockRequest("GET", "/images/json?"+v.Encode(), nil)
		if err != nil {
			t.Fatal(err)
		}
		var images []image
		if err := json.Unmarshal(b, &images); err != nil {
			t.Fatal(err)
		}

		return images
	}

	errMsg := "incorrect number of matches returned"
	if images := getImages("utest*/*"); len(images[0].RepoTags) != 2 {
		t.Fatal(errMsg)
	}
	if images := getImages("utest"); len(images[0].RepoTags) != 1 {
		t.Fatal(errMsg)
	}
	if images := getImages("utest*"); len(images[0].RepoTags) != 1 {
		t.Fatal(errMsg)
	}
	if images := getImages("*5000*/*"); len(images[0].RepoTags) != 1 {
		t.Fatal(errMsg)
	}

	logDone("images - filter param is applied")
}
