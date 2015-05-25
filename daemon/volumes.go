package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/pkg/chrootarchive"
	"github.com/tiborvass/docker/runconfig"
	"github.com/tiborvass/docker/volume"
	"github.com/docker/libcontainer/label"
)

type mountPoint struct {
	Name        string
	Destination string
	Driver      string
	RW          bool
	Volume      volume.Volume `json:"-"`
	Source      string
}

func (m *mountPoint) Setup() (string, error) {
	if m.Volume != nil {
		return m.Volume.Mount()
	}

	if len(m.Source) > 0 {
		if _, err := os.Stat(m.Source); err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
			if err := os.MkdirAll(m.Source, 0755); err != nil {
				return "", err
			}
		}
		return m.Source, nil
	}

	return "", fmt.Errorf("Unable to setup mount point, neither source nor volume defined")
}

func (m *mountPoint) Path() string {
	if m.Volume != nil {
		return m.Volume.Path()
	}

	return m.Source
}

func parseBindMount(spec string, mountLabel string, config *runconfig.Config) (*mountPoint, error) {
	bind := &mountPoint{
		RW: true,
	}
	arr := strings.Split(spec, ":")

	switch len(arr) {
	case 2:
		bind.Destination = arr[1]
	case 3:
		bind.Destination = arr[1]
		mode := arr[2]
		if !validMountMode(mode) {
			return nil, fmt.Errorf("invalid mode for volumes-from: %s", mode)
		}
		bind.RW = rwModes[mode]
		// check if we need to apply a SELinux label
		if strings.ContainsAny(mode, "zZ") {
			if err := label.Relabel(bind.Source, mountLabel, mode); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("Invalid volume specification: %s", spec)
	}

	name, source, err := parseVolumeSource(arr[0], config)
	if err != nil {
		return nil, err
	}

	if len(source) == 0 {
		bind.Driver = config.VolumeDriver
		if len(bind.Driver) == 0 {
			bind.Driver = volume.DefaultDriverName
		}
	} else {
		bind.Source = filepath.Clean(source)
	}

	bind.Name = name
	bind.Destination = filepath.Clean(bind.Destination)
	return bind, nil
}

func parseVolumesFrom(spec string) (string, string, error) {
	if len(spec) == 0 {
		return "", "", fmt.Errorf("malformed volumes-from specification: %s", spec)
	}

	specParts := strings.SplitN(spec, ":", 2)
	id := specParts[0]
	mode := "rw"

	if len(specParts) == 2 {
		mode = specParts[1]
		if !validMountMode(mode) {
			return "", "", fmt.Errorf("invalid mode for volumes-from: %s", mode)
		}
	}
	return id, mode, nil
}

// read-write modes
var rwModes = map[string]bool{
	"rw":   true,
	"rw,Z": true,
	"rw,z": true,
	"z,rw": true,
	"Z,rw": true,
	"Z":    true,
	"z":    true,
}

// read-only modes
var roModes = map[string]bool{
	"ro":   true,
	"ro,Z": true,
	"ro,z": true,
	"z,ro": true,
	"Z,ro": true,
}

func validMountMode(mode string) bool {
	return roModes[mode] || rwModes[mode]
}

func copyExistingContents(source, destination string) error {
	volList, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}
	if len(volList) > 0 {
		srcList, err := ioutil.ReadDir(destination)
		if err != nil {
			return err
		}
		if len(srcList) == 0 {
			// If the source volume is empty copy files from the root into the volume
			if err := chrootarchive.CopyWithTar(source, destination); err != nil {
				return err
			}
		}
	}
	return copyOwnership(source, destination)
}

// registerMountPoints initializes the container mount points with the configured volumes and bind mounts.
// It follows the next sequence to decide what to mount in each final destination:
//
// 1. Select the previously configured mount points for the containers, if any.
// 2. Select the volumes mounted from another containers. Overrides previously configured mount point destination.
// 3. Select the bind mounts set by the client. Overrides previously configured mount point destinations.
func (daemon *Daemon) registerMountPoints(container *Container, hostConfig *runconfig.HostConfig) error {
	binds := map[string]bool{}
	mountPoints := map[string]*mountPoint{}

	// 1. Read already configured mount points.
	for name, point := range container.MountPoints {
		mountPoints[name] = point
	}

	// 2. Read volumes from other containers.
	for _, v := range hostConfig.VolumesFrom {
		containerID, mode, err := parseVolumesFrom(v)
		if err != nil {
			return err
		}

		c, err := daemon.Get(containerID)
		if err != nil {
			return err
		}

		for _, m := range c.MountPoints {
			cp := m
			cp.RW = m.RW && mode != "ro"

			if len(m.Source) == 0 {
				v, err := createVolume(m.Name, m.Driver)
				if err != nil {
					return err
				}
				cp.Volume = v
			}

			mountPoints[cp.Destination] = cp
		}
	}

	// lock for labels
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// 3. Read bind mounts
	for _, b := range hostConfig.Binds {
		// #10618
		bind, err := parseBindMount(b, container.MountLabel, container.Config)
		if err != nil {
			return err
		}

		if binds[bind.Destination] {
			return fmt.Errorf("Duplicate bind mount %s", bind.Destination)
		}

		if len(bind.Name) > 0 && len(bind.Driver) > 0 {
			// set the label
			if err := label.SetFileCreateLabel(container.MountLabel); err != nil {
				return fmt.Errorf("Unable to setup default labeling for volume creation %s: %v", bind.Source, err)
			}

			// create the volume
			v, err := createVolume(bind.Name, bind.Driver)
			if err != nil {
				// reset the label
				if e := label.SetFileCreateLabel(""); e != nil {
					logrus.Errorf("Unable to reset labeling for volume creation %s: %v", bind.Source, e)
				}
				return err
			}
			bind.Volume = v

			// reset the label
			if err := label.SetFileCreateLabel(""); err != nil {
				return fmt.Errorf("Unable to reset labeling for volume creation %s: %v", bind.Source, err)
			}
		}

		binds[bind.Destination] = true
		mountPoints[bind.Destination] = bind
	}

	container.MountPoints = mountPoints

	return nil
}

// verifyOldVolumesInfo ports volumes configured for the containers pre docker 1.7.
// It reads the container configuration and creates valid mount points for the old volumes.
func (daemon *Daemon) verifyOldVolumesInfo(container *Container) error {
	jsonPath, err := container.jsonPath()
	if err != nil {
		return err
	}
	f, err := os.Open(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type oldContVolCfg struct {
		Volumes   map[string]string
		VolumesRW map[string]bool
	}

	vols := oldContVolCfg{
		Volumes:   make(map[string]string),
		VolumesRW: make(map[string]bool),
	}
	if err := json.NewDecoder(f).Decode(&vols); err != nil {
		return err
	}

	for destination, hostPath := range vols.Volumes {
		vfsPath := filepath.Join(daemon.root, "vfs", "dir")

		if strings.HasPrefix(hostPath, vfsPath) {
			id := filepath.Base(hostPath)

			rw := vols.VolumesRW != nil && vols.VolumesRW[destination]
			container.addLocalMountPoint(id, destination, rw)
		}
	}

	return container.ToDisk()
}

func createVolume(name, driverName string) (volume.Volume, error) {
	vd, err := getVolumeDriver(driverName)
	if err != nil {
		return nil, err
	}
	return vd.Create(name)
}

func removeVolume(v volume.Volume) error {
	vd, err := getVolumeDriver(v.DriverName())
	if err != nil {
		return nil
	}
	return vd.Remove(v)
}
