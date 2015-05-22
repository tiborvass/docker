package daemon

import (
	"testing"

	"github.com/tiborvass/docker/runconfig"
)

func TestParseBindMount(t *testing.T) {
	cases := []struct {
		bind      string
		driver    string
		expDest   string
		expSource string
		expName   string
		expDriver string
		expRW     bool
		fail      bool
	}{
		{"/tmp:/tmp", "", "/tmp", "/tmp", "", "", true, false},
		{"/tmp:/tmp:ro", "", "/tmp", "/tmp", "", "", false, false},
		{"/tmp:/tmp:rw", "", "/tmp", "/tmp", "", "", true, false},
		{"/tmp:/tmp:foo", "", "/tmp", "/tmp", "", "", false, true},
		{"name:/tmp", "", "", "", "", "", false, true},
		{"name:/tmp", "external", "/tmp", "", "name", "external", true, true},
		{"external/name:/tmp:rw", "", "/tmp", "", "name", "external", true, true},
		{"external/name:/tmp:ro", "", "/tmp", "", "name", "external", false, true},
		{"external/name:/tmp:foo", "", "/tmp", "", "name", "external", false, true},
		{"name:/tmp", "local", "", "", "", "", false, true},
		{"local/name:/tmp:rw", "", "", "", "", "", true, true},
	}

	for _, c := range cases {
		conf := &runconfig.Config{VolumeDriver: c.driver}
		m, err := parseBindMount(c.bind, conf)
		if c.fail {
			if err == nil {
				t.Fatalf("Expected error, was nil, for spec %s\n", c.bind)
			}
			continue
		}

		if m.Destination != c.expDest {
			t.Fatalf("Expected destination %s, was %s, for spec %s\n", c.expDest, m.Destination, c.bind)
		}

		if m.Source != c.expSource {
			t.Fatalf("Expected source %s, was %s, for spec %s\n", c.expSource, m.Source, c.bind)
		}

		if m.Name != c.expName {
			t.Fatalf("Expected name %s, was %s for spec %s\n", c.expName, m.Name, c.bind)
		}

		if m.Driver != c.expDriver {
			t.Fatalf("Expected driver %s, was %s, for spec %s\n", c.expDriver, m.Driver, c.bind)
		}

		if m.RW != c.expRW {
			t.Fatalf("Expected RW %v, was %v for spec %s\n", c.expRW, m.RW, c.bind)
		}
	}
}

func TestParseVolumeFrom(t *testing.T) {
	cases := []struct {
		spec    string
		expId   string
		expMode string
		fail    bool
	}{
		{"", "", "", true},
		{"foobar", "foobar", "rw", false},
		{"foobar:rw", "foobar", "rw", false},
		{"foobar:ro", "foobar", "ro", false},
		{"foobar:baz", "", "", true},
	}

	for _, c := range cases {
		id, mode, err := parseVolumesFrom(c.spec)
		if c.fail {
			if err == nil {
				t.Fatalf("Expected error, was nil, for spec %s\n", c.spec)
			}
			continue
		}

		if id != c.expId {
			t.Fatalf("Expected id %s, was %s, for spec %s\n", c.expId, id, c.spec)
		}
		if mode != c.expMode {
			t.Fatalf("Expected mode %s, was %s for spec %s\n", c.expMode, mode, c.spec)
		}
	}
}
