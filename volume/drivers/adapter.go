package volumedrivers

import (
	"github.com/tiborvass/docker/pkg/plugins"
	"github.com/tiborvass/docker/volume"
)

type volumeDriverAdapter struct {
	name  string
	proxy *volumeDriverProxy
}

func (a *volumeDriverAdapter) Name() string {
	return a.name
}

func (a *volumeDriverAdapter) Create(name string, opts map[string]string) (volume.Volume, error) {
	// First try a Get. For drivers that support Get this will return any
	// existing volume.
	v, err := a.proxy.Get(name)
	if v != nil {
		return &volumeAdapter{
			proxy:      a.proxy,
			name:       v.Name,
			driverName: a.Name(),
			eMount:     v.Mountpoint,
		}, nil
	}

	// Driver didn't support Get or volume didn't exist. Perform Create.
	err = a.proxy.Create(name, opts)
	if err != nil {
		return nil, err
	}
	return &volumeAdapter{
		proxy:      a.proxy,
		name:       name,
		driverName: a.name}, nil
}

func (a *volumeDriverAdapter) Remove(v volume.Volume) error {
	return a.proxy.Remove(v.Name())
}

func (a *volumeDriverAdapter) List() ([]volume.Volume, error) {
	ls, err := a.proxy.List()
	if err != nil {
		return nil, err
	}

	var out []volume.Volume
	for _, vp := range ls {
		out = append(out, &volumeAdapter{
			proxy:      a.proxy,
			name:       vp.Name,
			driverName: a.name,
			eMount:     vp.Mountpoint,
		})
	}
	return out, nil
}

func (a *volumeDriverAdapter) Get(name string) (volume.Volume, error) {
	v, err := a.proxy.Get(name)
	if err != nil {
		// TODO: remove this hack. Allows back compat with volume drivers that don't support this call
		if !plugins.IsNotFound(err) {
			return nil, err
		}
		return a.Create(name, nil)
	}

	return &volumeAdapter{
		proxy:      a.proxy,
		name:       v.Name,
		driverName: a.Name(),
		eMount:     v.Mountpoint,
	}, nil
}

type volumeAdapter struct {
	proxy      *volumeDriverProxy
	name       string
	driverName string
	eMount     string // ephemeral host volume path
}

type proxyVolume struct {
	Name       string
	Mountpoint string
}

func (a *volumeAdapter) Name() string {
	return a.name
}

func (a *volumeAdapter) DriverName() string {
	return a.driverName
}

func (a *volumeAdapter) Path() string {
	if len(a.eMount) > 0 {
		return a.eMount
	}
	m, _ := a.proxy.Path(a.name)
	return m
}

func (a *volumeAdapter) Mount() (string, error) {
	var err error
	a.eMount, err = a.proxy.Mount(a.name)
	return a.eMount, err
}

func (a *volumeAdapter) Unmount() error {
	return a.proxy.Unmount(a.name)
}
