// +build !windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/integration-cli/checker"
	"github.com/tiborvass/docker/integration-cli/daemon"
	testdaemon "github.com/tiborvass/docker/internal/test/daemon"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/tiborvass/docker/volume"
	"github.com/go-check/check"
	"gotest.tools/assert"
)

const volumePluginName = "test-external-volume-driver"

func init() {
	check.Suite(&DockerExternalVolumeSuite{
		ds: &DockerSuite{},
	})
}

type eventCounter struct {
	activations int
	creations   int
	removals    int
	mounts      int
	unmounts    int
	paths       int
	lists       int
	gets        int
	caps        int
}

type DockerExternalVolumeSuite struct {
	ds *DockerSuite
	d  *daemon.Daemon
	*volumePlugin
}

func (s *DockerExternalVolumeSuite) SetUpTest(c *check.C) {
	testRequires(c, testEnv.IsLocalDaemon)
	s.d = daemon.New(c, dockerBinary, dockerdBinary, testdaemon.WithEnvironment(testEnv.Execution))
	s.ec = &eventCounter{}
}

func (s *DockerExternalVolumeSuite) TearDownTest(c *check.C) {
	if s.d != nil {
		s.d.Stop(c)
		s.ds.TearDownTest(c)
	}
}

func (s *DockerExternalVolumeSuite) SetUpSuite(c *check.C) {
	s.volumePlugin = newVolumePlugin(c, volumePluginName)
}

type volumePlugin struct {
	ec *eventCounter
	*httptest.Server
	vols map[string]vol
}

type vol struct {
	Name       string
	Mountpoint string
	Ninja      bool // hack used to trigger a null volume return on `Get`
	Status     map[string]interface{}
	Options    map[string]string
}

func (p *volumePlugin) Close() {
	p.Server.Close()
}

func newVolumePlugin(c *check.C, name string) *volumePlugin {
	mux := http.NewServeMux()
	s := &volumePlugin{Server: httptest.NewServer(mux), ec: &eventCounter{}, vols: make(map[string]vol)}

	type pluginRequest struct {
		Name string
		Opts map[string]string
		ID   string
	}

	type pluginResp struct {
		Mountpoint string `json:",omitempty"`
		Err        string `json:",omitempty"`
	}

	read := func(b io.ReadCloser) (pluginRequest, error) {
		defer b.Close()
		var pr pluginRequest
		err := json.NewDecoder(b).Decode(&pr)
		return pr, err
	}

	send := func(w http.ResponseWriter, data interface{}) {
		switch t := data.(type) {
		case error:
			http.Error(w, t.Error(), 500)
		case string:
			w.Header().Set("Content-Type", "application/vnd.docker.plugins.v1+json")
			fmt.Fprintln(w, t)
		default:
			w.Header().Set("Content-Type", "application/vnd.docker.plugins.v1+json")
			json.NewEncoder(w).Encode(&data)
		}
	}

	mux.HandleFunc("/Plugin.Activate", func(w http.ResponseWriter, r *http.Request) {
		s.ec.activations++
		send(w, `{"Implements": ["VolumeDriver"]}`)
	})

	mux.HandleFunc("/VolumeDriver.Create", func(w http.ResponseWriter, r *http.Request) {
		s.ec.creations++
		pr, err := read(r.Body)
		if err != nil {
			send(w, err)
			return
		}
		_, isNinja := pr.Opts["ninja"]
		status := map[string]interface{}{"Hello": "world"}
		s.vols[pr.Name] = vol{Name: pr.Name, Ninja: isNinja, Status: status, Options: pr.Opts}
		send(w, nil)
	})

	mux.HandleFunc("/VolumeDriver.List", func(w http.ResponseWriter, r *http.Request) {
		s.ec.lists++
		vols := make([]vol, 0, len(s.vols))
		for _, v := range s.vols {
			if v.Ninja {
				continue
			}
			vols = append(vols, v)
		}
		send(w, map[string][]vol{"Volumes": vols})
	})

	mux.HandleFunc("/VolumeDriver.Get", func(w http.ResponseWriter, r *http.Request) {
		s.ec.gets++
		pr, err := read(r.Body)
		if err != nil {
			send(w, err)
			return
		}

		v, exists := s.vols[pr.Name]
		if !exists {
			send(w, `{"Err": "no such volume"}`)
		}

		if v.Ninja {
			send(w, map[string]vol{})
			return
		}

		v.Mountpoint = hostVolumePath(pr.Name)
		send(w, map[string]vol{"Volume": v})
		return
	})

	mux.HandleFunc("/VolumeDriver.Remove", func(w http.ResponseWriter, r *http.Request) {
		s.ec.removals++
		pr, err := read(r.Body)
		if err != nil {
			send(w, err)
			return
		}

		v, ok := s.vols[pr.Name]
		if !ok {
			send(w, nil)
			return
		}

		if err := os.RemoveAll(hostVolumePath(v.Name)); err != nil {
			send(w, &pluginResp{Err: err.Error()})
			return
		}
		delete(s.vols, v.Name)
		send(w, nil)
	})

	mux.HandleFunc("/VolumeDriver.Path", func(w http.ResponseWriter, r *http.Request) {
		s.ec.paths++

		pr, err := read(r.Body)
		if err != nil {
			send(w, err)
			return
		}
		p := hostVolumePath(pr.Name)
		send(w, &pluginResp{Mountpoint: p})
	})

	mux.HandleFunc("/VolumeDriver.Mount", func(w http.ResponseWriter, r *http.Request) {
		s.ec.mounts++

		pr, err := read(r.Body)
		if err != nil {
			send(w, err)
			return
		}

		if v, exists := s.vols[pr.Name]; exists {
			// Use this to simulate a mount failure
			if _, exists := v.Options["invalidOption"]; exists {
				send(w, fmt.Errorf("invalid argument"))
				return
			}
		}

		p := hostVolumePath(pr.Name)
		if err := os.MkdirAll(p, 0755); err != nil {
			send(w, &pluginResp{Err: err.Error()})
			return
		}

		if err := ioutil.WriteFile(filepath.Join(p, "test"), []byte(s.Server.URL), 0644); err != nil {
			send(w, err)
			return
		}

		if err := ioutil.WriteFile(filepath.Join(p, "mountID"), []byte(pr.ID), 0644); err != nil {
			send(w, err)
			return
		}

		send(w, &pluginResp{Mountpoint: p})
	})

	mux.HandleFunc("/VolumeDriver.Unmount", func(w http.ResponseWriter, r *http.Request) {
		s.ec.unmounts++

		_, err := read(r.Body)
		if err != nil {
			send(w, err)
			return
		}

		send(w, nil)
	})

	mux.HandleFunc("/VolumeDriver.Capabilities", func(w http.ResponseWriter, r *http.Request) {
		s.ec.caps++

		_, err := read(r.Body)
		if err != nil {
			send(w, err)
			return
		}

		send(w, `{"Capabilities": { "Scope": "global" }}`)
	})

	err := os.MkdirAll("/etc/docker/plugins", 0755)
	assert.NilError(c, err)

	err = ioutil.WriteFile("/etc/docker/plugins/"+name+".spec", []byte(s.Server.URL), 0644)
	assert.NilError(c, err)
	return s
}

func (s *DockerExternalVolumeSuite) TearDownSuite(c *check.C) {
	s.volumePlugin.Close()

	err := os.RemoveAll("/etc/docker/plugins")
	assert.NilError(c, err)
}

func (s *DockerExternalVolumeSuite) TestVolumeCLICreateOptionConflict(c *check.C) {
	dockerCmd(c, "volume", "create", "test")

	out, _, err := dockerCmdWithError("volume", "create", "test", "--driver", volumePluginName)
	assert.Assert(c, err, check.NotNil, check.Commentf("volume create exception name already in use with another driver"))
	assert.Assert(c, out, checker.Contains, "must be unique")

	out, _ = dockerCmd(c, "volume", "inspect", "--format={{ .Driver }}", "test")
	_, _, err = dockerCmdWithError("volume", "create", "test", "--driver", strings.TrimSpace(out))
	assert.NilError(c, err)
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverNamed(c *check.C) {
	s.d.StartWithBusybox(c)

	out, err := s.d.Cmd("run", "--rm", "--name", "test-data", "-v", "external-volume-test:/tmp/external-volume-test", "--volume-driver", volumePluginName, "busybox:latest", "cat", "/tmp/external-volume-test/test")
	assert.NilError(c, err, out)
	assert.Assert(c, out, checker.Contains, s.Server.URL)

	_, err = s.d.Cmd("volume", "rm", "external-volume-test")
	assert.NilError(c, err)

	p := hostVolumePath("external-volume-test")
	_, err = os.Lstat(p)
	assert.ErrorContains(c, err, "")
	assert.Assert(c, os.IsNotExist(err), checker.True, check.Commentf("Expected volume path in host to not exist: %s, %v\n", p, err))

	assert.Assert(c, s.ec.activations, checker.Equals, 1)
	assert.Assert(c, s.ec.creations, checker.Equals, 1)
	assert.Assert(c, s.ec.removals, checker.Equals, 1)
	assert.Assert(c, s.ec.mounts, checker.Equals, 1)
	assert.Assert(c, s.ec.unmounts, checker.Equals, 1)
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverUnnamed(c *check.C) {
	s.d.StartWithBusybox(c)

	out, err := s.d.Cmd("run", "--rm", "--name", "test-data", "-v", "/tmp/external-volume-test", "--volume-driver", volumePluginName, "busybox:latest", "cat", "/tmp/external-volume-test/test")
	assert.NilError(c, err, out)
	assert.Assert(c, out, checker.Contains, s.Server.URL)

	assert.Assert(c, s.ec.activations, checker.Equals, 1)
	assert.Assert(c, s.ec.creations, checker.Equals, 1)
	assert.Assert(c, s.ec.removals, checker.Equals, 1)
	assert.Assert(c, s.ec.mounts, checker.Equals, 1)
	assert.Assert(c, s.ec.unmounts, checker.Equals, 1)
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverVolumesFrom(c *check.C) {
	s.d.StartWithBusybox(c)

	out, err := s.d.Cmd("run", "--name", "vol-test1", "-v", "/foo", "--volume-driver", volumePluginName, "busybox:latest")
	assert.NilError(c, err, out)

	out, err = s.d.Cmd("run", "--rm", "--volumes-from", "vol-test1", "--name", "vol-test2", "busybox", "ls", "/tmp")
	assert.NilError(c, err, out)

	out, err = s.d.Cmd("rm", "-fv", "vol-test1")
	assert.NilError(c, err, out)

	assert.Assert(c, s.ec.activations, checker.Equals, 1)
	assert.Assert(c, s.ec.creations, checker.Equals, 1)
	assert.Assert(c, s.ec.removals, checker.Equals, 1)
	assert.Assert(c, s.ec.mounts, checker.Equals, 2)
	assert.Assert(c, s.ec.unmounts, checker.Equals, 2)
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverDeleteContainer(c *check.C) {
	s.d.StartWithBusybox(c)

	out, err := s.d.Cmd("run", "--name", "vol-test1", "-v", "/foo", "--volume-driver", volumePluginName, "busybox:latest")
	assert.NilError(c, err, out)

	out, err = s.d.Cmd("rm", "-fv", "vol-test1")
	assert.NilError(c, err, out)

	assert.Assert(c, s.ec.activations, checker.Equals, 1)
	assert.Assert(c, s.ec.creations, checker.Equals, 1)
	assert.Assert(c, s.ec.removals, checker.Equals, 1)
	assert.Assert(c, s.ec.mounts, checker.Equals, 1)
	assert.Assert(c, s.ec.unmounts, checker.Equals, 1)
}

func hostVolumePath(name string) string {
	return fmt.Sprintf("/var/lib/docker/volumes/%s", name)
}

// Make sure a request to use a down driver doesn't block other requests
func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverLookupNotBlocked(c *check.C) {
	specPath := "/etc/docker/plugins/down-driver.spec"
	err := ioutil.WriteFile(specPath, []byte("tcp://127.0.0.7:9999"), 0644)
	assert.NilError(c, err)
	defer os.RemoveAll(specPath)

	chCmd1 := make(chan struct{})
	chCmd2 := make(chan error)
	cmd1 := exec.Command(dockerBinary, "volume", "create", "-d", "down-driver")
	cmd2 := exec.Command(dockerBinary, "volume", "create")

	assert.Assert(c, cmd1.Start(), checker.IsNil)
	defer cmd1.Process.Kill()
	time.Sleep(100 * time.Millisecond) // ensure API has been called
	assert.Assert(c, cmd2.Start(), checker.IsNil)

	go func() {
		cmd1.Wait()
		close(chCmd1)
	}()
	go func() {
		chCmd2 <- cmd2.Wait()
	}()

	select {
	case <-chCmd1:
		cmd2.Process.Kill()
		c.Fatalf("volume create with down driver finished unexpectedly")
	case err := <-chCmd2:
		assert.NilError(c, err)
	case <-time.After(5 * time.Second):
		cmd2.Process.Kill()
		c.Fatal("volume creates are blocked by previous create requests when previous driver is down")
	}
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverRetryNotImmediatelyExists(c *check.C) {
	s.d.StartWithBusybox(c)
	driverName := "test-external-volume-driver-retry"

	errchan := make(chan error)
	started := make(chan struct{})
	go func() {
		close(started)
		if out, err := s.d.Cmd("run", "--rm", "--name", "test-data-retry", "-v", "external-volume-test:/tmp/external-volume-test", "--volume-driver", driverName, "busybox:latest"); err != nil {
			errchan <- fmt.Errorf("%v:\n%s", err, out)
		}
		close(errchan)
	}()

	<-started
	// wait for a retry to occur, then create spec to allow plugin to register
	time.Sleep(2 * time.Second)
	p := newVolumePlugin(c, driverName)
	defer p.Close()

	select {
	case err := <-errchan:
		assert.NilError(c, err)
	case <-time.After(8 * time.Second):
		c.Fatal("volume creates fail when plugin not immediately available")
	}

	_, err := s.d.Cmd("volume", "rm", "external-volume-test")
	assert.NilError(c, err)

	assert.Assert(c, p.ec.activations, checker.Equals, 1)
	assert.Assert(c, p.ec.creations, checker.Equals, 1)
	assert.Assert(c, p.ec.removals, checker.Equals, 1)
	assert.Assert(c, p.ec.mounts, checker.Equals, 1)
	assert.Assert(c, p.ec.unmounts, checker.Equals, 1)
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverBindExternalVolume(c *check.C) {
	dockerCmd(c, "volume", "create", "-d", volumePluginName, "foo")
	dockerCmd(c, "run", "-d", "--name", "testing", "-v", "foo:/bar", "busybox", "top")

	var mounts []struct {
		Name   string
		Driver string
	}
	out := inspectFieldJSON(c, "testing", "Mounts")
	assert.Assert(c, json.NewDecoder(strings.NewReader(out)).Decode(&mounts), checker.IsNil)
	assert.Assert(c, len(mounts), checker.Equals, 1, check.Commentf("%s", out))
	assert.Assert(c, mounts[0].Name, checker.Equals, "foo")
	assert.Assert(c, mounts[0].Driver, checker.Equals, volumePluginName)
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverList(c *check.C) {
	dockerCmd(c, "volume", "create", "-d", volumePluginName, "abc3")
	out, _ := dockerCmd(c, "volume", "ls")
	ls := strings.Split(strings.TrimSpace(out), "\n")
	assert.Assert(c, len(ls), check.Equals, 2, check.Commentf("\n%s", out))

	vol := strings.Fields(ls[len(ls)-1])
	assert.Assert(c, len(vol), check.Equals, 2, check.Commentf("%v", vol))
	assert.Assert(c, vol[0], check.Equals, volumePluginName)
	assert.Assert(c, vol[1], check.Equals, "abc3")

	assert.Assert(c, s.ec.lists, check.Equals, 1)
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverGet(c *check.C) {
	out, _, err := dockerCmdWithError("volume", "inspect", "dummy")
	assert.ErrorContains(c, err, "", out)
	assert.Assert(c, out, checker.Contains, "No such volume")
	assert.Assert(c, s.ec.gets, check.Equals, 1)

	dockerCmd(c, "volume", "create", "test", "-d", volumePluginName)
	out, _ = dockerCmd(c, "volume", "inspect", "test")

	type vol struct {
		Status map[string]string
	}
	var st []vol

	assert.Assert(c, json.Unmarshal([]byte(out), &st), checker.IsNil)
	assert.Assert(c, st, checker.HasLen, 1)
	assert.Assert(c, st[0].Status, checker.HasLen, 1, check.Commentf("%v", st[0]))
	assert.Assert(c, st[0].Status["Hello"], checker.Equals, "world", check.Commentf("%v", st[0].Status))
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverWithDaemonRestart(c *check.C) {
	dockerCmd(c, "volume", "create", "-d", volumePluginName, "abc1")
	s.d.Restart(c)

	dockerCmd(c, "run", "--name=test", "-v", "abc1:/foo", "busybox", "true")
	var mounts []types.MountPoint
	inspectFieldAndUnmarshall(c, "test", "Mounts", &mounts)
	assert.Assert(c, mounts, checker.HasLen, 1)
	assert.Assert(c, mounts[0].Driver, checker.Equals, volumePluginName)
}

// Ensures that the daemon handles when the plugin responds to a `Get` request with a null volume and a null error.
// Prior the daemon would panic in this scenario.
func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverGetEmptyResponse(c *check.C) {
	s.d.Start(c)

	out, err := s.d.Cmd("volume", "create", "-d", volumePluginName, "abc2", "--opt", "ninja=1")
	assert.NilError(c, err, out)

	out, err = s.d.Cmd("volume", "inspect", "abc2")
	assert.ErrorContains(c, err, "", out)
	assert.Assert(c, out, checker.Contains, "No such volume")
}

// Ensure only cached paths are used in volume list to prevent N+1 calls to `VolumeDriver.Path`
//
// TODO(@cpuguy83): This test is testing internal implementation. In all the cases here, there may not even be a path
// 	available because the volume is not even mounted. Consider removing this test.
func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverPathCalls(c *check.C) {
	s.d.Start(c)
	assert.Assert(c, s.ec.paths, checker.Equals, 0)

	out, err := s.d.Cmd("volume", "create", "test", "--driver=test-external-volume-driver")
	assert.NilError(c, err, out)
	assert.Assert(c, s.ec.paths, checker.Equals, 0)

	out, err = s.d.Cmd("volume", "ls")
	assert.NilError(c, err, out)
	assert.Assert(c, s.ec.paths, checker.Equals, 0)
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverMountID(c *check.C) {
	s.d.StartWithBusybox(c)

	out, err := s.d.Cmd("run", "--rm", "-v", "external-volume-test:/tmp/external-volume-test", "--volume-driver", volumePluginName, "busybox:latest", "cat", "/tmp/external-volume-test/test")
	assert.NilError(c, err, out)
	assert.Assert(c, strings.TrimSpace(out) != "")
}

// Check that VolumeDriver.Capabilities gets called, and only called once
func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverCapabilities(c *check.C) {
	s.d.Start(c)
	assert.Assert(c, s.ec.caps, checker.Equals, 0)

	for i := 0; i < 3; i++ {
		out, err := s.d.Cmd("volume", "create", "-d", volumePluginName, fmt.Sprintf("test%d", i))
		assert.NilError(c, err, out)
		assert.Assert(c, s.ec.caps, checker.Equals, 1)
		out, err = s.d.Cmd("volume", "inspect", "--format={{.Scope}}", fmt.Sprintf("test%d", i))
		assert.NilError(c, err)
		assert.Equal(c, strings.TrimSpace(out), volume.GlobalScope)
	}
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverOutOfBandDelete(c *check.C) {
	driverName := stringid.GenerateRandomID()
	p := newVolumePlugin(c, driverName)
	defer p.Close()

	s.d.StartWithBusybox(c)

	out, err := s.d.Cmd("volume", "create", "-d", driverName, "--name", "test")
	assert.NilError(c, err, out)

	out, err = s.d.Cmd("volume", "create", "-d", "local", "--name", "test")
	assert.ErrorContains(c, err, "", out)
	assert.Assert(c, out, checker.Contains, "must be unique")

	// simulate out of band volume deletion on plugin level
	delete(p.vols, "test")

	// test re-create with same driver
	out, err = s.d.Cmd("volume", "create", "-d", driverName, "--opt", "foo=bar", "--name", "test")
	assert.NilError(c, err, out)
	out, err = s.d.Cmd("volume", "inspect", "test")
	assert.NilError(c, err, out)

	var vs []types.Volume
	err = json.Unmarshal([]byte(out), &vs)
	assert.NilError(c, err)
	assert.Assert(c, vs, checker.HasLen, 1)
	assert.Assert(c, vs[0].Driver, checker.Equals, driverName)
	assert.Assert(c, vs[0].Options, checker.NotNil)
	assert.Assert(c, vs[0].Options["foo"], checker.Equals, "bar")
	assert.Assert(c, vs[0].Driver, checker.Equals, driverName)

	// simulate out of band volume deletion on plugin level
	delete(p.vols, "test")

	// test create with different driver
	out, err = s.d.Cmd("volume", "create", "-d", "local", "--name", "test")
	assert.NilError(c, err, out)

	out, err = s.d.Cmd("volume", "inspect", "test")
	assert.NilError(c, err, out)
	vs = nil
	err = json.Unmarshal([]byte(out), &vs)
	assert.NilError(c, err)
	assert.Assert(c, vs, checker.HasLen, 1)
	assert.Assert(c, vs[0].Options, checker.HasLen, 0)
	assert.Assert(c, vs[0].Driver, checker.Equals, "local")
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverUnmountOnMountFail(c *check.C) {
	s.d.StartWithBusybox(c)
	s.d.Cmd("volume", "create", "-d", "test-external-volume-driver", "--opt=invalidOption=1", "--name=testumount")

	out, _ := s.d.Cmd("run", "-v", "testumount:/foo", "busybox", "true")
	assert.Assert(c, s.ec.unmounts, checker.Equals, 0, check.Commentf("%s", out))
	out, _ = s.d.Cmd("run", "-w", "/foo", "-v", "testumount:/foo", "busybox", "true")
	assert.Assert(c, s.ec.unmounts, checker.Equals, 0, check.Commentf("%s", out))
}

func (s *DockerExternalVolumeSuite) TestExternalVolumeDriverUnmountOnCp(c *check.C) {
	s.d.StartWithBusybox(c)
	s.d.Cmd("volume", "create", "-d", "test-external-volume-driver", "--name=test")

	out, _ := s.d.Cmd("run", "-d", "--name=test", "-v", "test:/foo", "busybox", "/bin/sh", "-c", "touch /test && top")
	assert.Assert(c, s.ec.mounts, checker.Equals, 1, check.Commentf("%s", out))

	out, _ = s.d.Cmd("cp", "test:/test", "/tmp/test")
	assert.Assert(c, s.ec.mounts, checker.Equals, 2, check.Commentf("%s", out))
	assert.Assert(c, s.ec.unmounts, checker.Equals, 1, check.Commentf("%s", out))

	out, _ = s.d.Cmd("kill", "test")
	assert.Assert(c, s.ec.unmounts, checker.Equals, 2, check.Commentf("%s", out))
}
