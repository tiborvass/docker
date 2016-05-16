package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/libcontainerd"
	"github.com/docker/docker/oci"
	"github.com/docker/docker/pkg/plugins"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/system"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/restartmanager"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/opencontainers/specs/specs-go"
)

var manager *Manager

type ErrNotFound string

func (name ErrNotFound) Error() string { return fmt.Sprintf("plugin %q not found", string(name)) }

type ErrInadequateCapability struct {
	name       string
	capability string
}

func (e ErrInadequateCapability) Error() string {
	return fmt.Sprintf("plugin %q found, but not with %q capability", e.name, e.capability)
}

type Plugin interface {
	Client() *plugins.Client
	Name() string
}

type plugin struct {
	//sync.RWMutex TODO
	p              types.Plugin
	client         *plugins.Client
	restartManager restartmanager.RestartManager
}

func (p *plugin) Client() *plugins.Client {
	return p.client
}

func (p *plugin) Name() string {
	return p.p.Name
}

func newPlugin(name string) *plugin {
	return &plugin{
		p: types.Plugin{
			Name: name,
		},
	}
}

// Manager controls the plugin subsystem.
type Manager struct {
	sync.RWMutex
	libRoot          string
	runRoot          string
	plugins          map[string]*plugin
	handlers         map[string]func(string, *plugins.Client)
	containerdClient libcontainerd.Client
	registryService  registry.Service
	handleLegacy     bool
}

func GetManager() *Manager {
	return manager
}

// NewManager instantiates a Manager
//func NewManager(root, execRoot string, remote libcontainerd.Remote) (*Manager, error) {
func Init(root, execRoot string, remote libcontainerd.Remote, rs registry.Service) (err error) {
	if manager != nil {
		return nil
	}

	root = filepath.Join(root, "plugins")
	execRoot = filepath.Join(execRoot, "plugins")
	for _, dir := range []string{root, execRoot} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}

	manager = &Manager{
		libRoot:         root,
		runRoot:         execRoot,
		plugins:         make(map[string]*plugin),
		handlers:        make(map[string]func(string, *plugins.Client)),
		registryService: rs,
		handleLegacy:    true,
	}
	if err := os.MkdirAll(manager.runRoot, 0700); err != nil {
		return err
	}
	if err := manager.init(); err != nil {
		return err
	}
	manager.containerdClient, err = remote.Client(manager)
	if err != nil {
		return err
	}
	return nil
}

func Handle(capability string, callback func(string, *plugins.Client)) {
	pluginType := fmt.Sprintf("docker.%s/1", strings.ToLower(capability))
	manager.handlers[pluginType] = callback
	if manager.handleLegacy {
		plugins.Handle(capability, callback)
	}
}

func (pm *Manager) get(name string) (*plugin, error) {
	pm.RLock()
	p, ok := pm.plugins[name]
	pm.RUnlock()
	if !ok {
		return nil, ErrNotFound(name)
	}
	return p, nil
}

func LookupWithCapability(name, capability string) (Plugin, error) {
	manager.RLock()
	p, ok := manager.plugins[name]
	manager.RUnlock()

	if !ok {
		if manager.handleLegacy {
			p, err := plugins.Get(name, capability)
			if err != nil {
				return nil, fmt.Errorf("legacy plugin: %v", err)
			}
			return p, nil
		}
		return nil, ErrNotFound(name)
	}
	capability = strings.ToLower(capability)
	for _, typ := range p.p.Manifest.Interface.Types {
		if typ.Capability == capability && typ.Prefix == "docker" {
			return p, nil
		}
	}
	return nil, ErrInadequateCapability{name, capability}
}

func FindWithCapability(capability string) ([]Plugin, error) {
	result := make([]Plugin, 0, 1)
	manager.RLock()
	defer manager.RUnlock()
pluginLoop:
	for _, p := range manager.plugins {
		for _, typ := range p.p.Manifest.Interface.Types {
			if typ.Capability != capability || typ.Prefix != "docker" {
				continue pluginLoop
			}
		}
		result = append(result, p)
	}
	if manager.handleLegacy {
		pl, err := plugins.GetAll(capability)
		if err != nil {
			return nil, fmt.Errorf("legacy plugin: %v", err)
		}
		for _, p := range pl {
			if _, ok := manager.plugins[p.Name()]; !ok {
				result = append(result, p)
			}
		}
	}
	return result, nil
}

// StateChanged updates daemon inter...
func (pm *Manager) StateChanged(id string, e libcontainerd.StateInfo) error {
	logrus.Debugf("plugin statechanged %s %#v", id, e)

	return nil
}

// AttachStreams attaches io streams to the plugin
func (pm *Manager) AttachStreams(id string, iop libcontainerd.IOPipe) error {
	iop.Stdin.Close()

	logger := logrus.New()
	logger.Hooks.Add(logHook{id})
	// TODO: cache writer per id
	w := logger.Writer()
	go func() {
		io.Copy(w, iop.Stdout)
	}()
	go func() {
		// TODO: update logrus and use logger.WriterLevel
		io.Copy(w, iop.Stderr)
	}()
	return nil
}

func (pm *Manager) init() error {
	dt, err := os.Open(filepath.Join(pm.libRoot, "plugins.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := json.NewDecoder(dt).Decode(&pm.plugins); err != nil {
		return err
	}
	// FIXME: validate, restore

	return nil
}

func (pm *Manager) enable(p *plugin) error {
	if privileges := computePrivileges(&p.p.Manifest); privileges != nil {
	}

	spec, err := pm.initSpec(p)
	if err != nil {
		return err
	}

	p.restartManager = restartmanager.New(container.RestartPolicy{Name: "always"}, 0)
	p.p.ID = stringid.GenerateNonCryptoID()
	if err := pm.containerdClient.Create(p.p.ID, libcontainerd.Spec(spec), libcontainerd.WithRestartManager(p.restartManager)); err != nil { // POC-only
		return err
	}

	// TODO: honor Socket from manifest
	p.client, err = plugins.NewClient("unix://"+filepath.Join(pm.runRoot, p.p.Name, p.p.Manifest.Interface.Socket), nil)
	if err != nil {
		return err
	}

	//TODO: check net.Dial

	pm.Lock() // fixme: lock single record
	p.p.Active = true
	pm.save()
	pm.Unlock()

	for _, typ := range p.p.Manifest.Interface.Types {
		if handler := pm.handlers[typ.String()]; handler != nil {
			handler(p.Name(), p.Client())
		}
	}

	return nil
}

func (pm *Manager) initPlugin(p *plugin) error {
	dt, err := os.Open(filepath.Join(pm.libRoot, p.p.Name, "manifest.json"))
	if err != nil {
		return err
	}
	err = json.NewDecoder(dt).Decode(&p.p.Manifest)
	dt.Close()
	if err != nil {
		return err
	}

	p.p.Config.Mounts = make([]types.PluginMount, len(p.p.Manifest.Mounts))
	for i, mount := range p.p.Manifest.Mounts {
		p.p.Config.Mounts[i] = mount.PluginMount
	}
	p.p.Config.Env = make([]string, len(p.p.Manifest.Env))
	for i, env := range p.p.Manifest.Env {
		if env.Value != nil {
			p.p.Config.Env[i] = fmt.Sprintf("%s=%s", env.Name, *env.Value)
		}
	}
	p.p.Config.Args = make([]string, len(p.p.Manifest.Args))
	for i, arg := range p.p.Manifest.Args {
		if arg.Value != nil {
			p.p.Config.Args[i] = fmt.Sprintf("%s=%s", arg.Name, *arg.Value)
		}
	}

	f, err := os.Create(filepath.Join(pm.libRoot, p.p.Name, "plugin-config.json"))
	if err != nil {
		return err
	}
	err = json.NewEncoder(f).Encode(&p.p.Config)
	f.Close()
	return err
}
func (pm *Manager) initSpec(p *plugin) (specs.Spec, error) {
	s := oci.DefaultSpec()
	rootfs := filepath.Join(pm.libRoot, p.p.Name, "rootfs")
	s.Root = specs.Root{
		Path:     rootfs,
		Readonly: false, // TODO: all plugins should be readonly?
	}
	pluginsRuntimeMounted := false
	for _, mount := range p.p.Config.Mounts {
		m := specs.Mount{
			Destination: mount.Destination,
			Type:        mount.Type,
			Options:     mount.Options,
		}
		// TODO: if nil, it's required and user didn't set it
		if mount.Source != nil {
			m.Source = *mount.Source
		}
		// TODO: remove docker.* types
		switch mount.Type {
		case "docker.plugin.runtime":
			//TODO: ensure m.Source is not set
			m.Source = filepath.Join(pm.runRoot, p.p.Name)
			m.Type = "bind"
			m.Options = []string{"rbind", "rprivate"}
			pluginsRuntimeMounted = true
		case "docker.plugin.state":
			//TODO: ensure m.Source is not set
			m.Source = filepath.Join(pm.libRoot, p.p.Name, "state")
			m.Type = "bind"
			m.Options = []string{"rbind", "rprivate"}
		}
		if m.Source != "" && m.Type == "bind" {
			// TODO: followsymlinkpath
			fi, err := os.Lstat(filepath.Join(rootfs, string(os.PathSeparator), m.Destination))
			if err != nil {
				return s, err
			}
			if fi.IsDir() {
				if err := os.MkdirAll(m.Source, 0700); err != nil {
					return s, err
				}
			}
		}
		s.Mounts = append(s.Mounts, m)
	}
	if !pluginsRuntimeMounted {
		return s, fmt.Errorf("plugin %q requires mount of type docker.plugins.runtime", p.p.Name)
	}

	envs := make([]string, 1, len(p.p.Config.Env)+1)
	envs[0] = "PATH=" + system.DefaultPathEnv
	envs = append(envs, p.p.Config.Env...)

	s.Process = specs.Process{
		Terminal: false,
		Args:     append(p.p.Manifest.Entrypoint, p.p.Config.Args...),
		Cwd:      "/", // TODO: add in manifest?
		Env:      envs,
	}
	return s, nil
}

func (pm *Manager) disable(p *plugin) error {
	if err := p.restartManager.Cancel(); err != nil {
		logrus.Error(err)
	}
	if err := pm.containerdClient.Signal(p.p.ID, int(syscall.SIGKILL)); err != nil {
		logrus.Error(err)
	}
	os.RemoveAll(filepath.Join(pm.runRoot, p.p.Name))
	pm.Lock() // fixme: lock single record
	defer pm.Unlock()
	p.p.ID = ""
	p.p.Active = false
	pm.save()
	return nil
}

func (pm *Manager) remove(p *plugin) error {
	if p.p.Active {
		return fmt.Errorf("plugin %s is active", p.p.Name)
	}
	pm.Lock() // fixme: lock single record
	defer pm.Unlock()
	os.RemoveAll(filepath.Join(pm.libRoot, p.p.Name))
	delete(pm.plugins, p.p.Name)
	pm.save()
	return nil
}

func (pm *Manager) set(p *plugin, args []string) error {
	m := make(map[string]string, len(args))
	for _, arg := range args {
		i := strings.Index(arg, "=")
		if i < 0 {
			return fmt.Errorf("No equal sign '=' found in %s", arg)
		}
		m[arg[:i]] = arg[i+1:]
	}
	return errors.New("not implemented")
}

// fixme: not safe
func (pm *Manager) save() error {
	f, err := os.Create(filepath.Join(pm.libRoot, "plugins.json"))
	if err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(pm.plugins)
}

type logHook struct{ id string }

func (logHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (l logHook) Fire(entry *logrus.Entry) error {
	entry.Data = logrus.Fields{"plugin": l.id}
	return nil
}

// Panics
func computePrivileges(m *types.PluginManifest) (privileges *types.PluginPrivileges) {
	ensureNotEmpty := func() {
		if privileges == nil {
			privileges = new(types.PluginPrivileges)
		}
	}
	if m.Network.Type != "null" && m.Network.Type != "bridge" {
		ensureNotEmpty()
		privileges.Network = &m.Network.Type
	}
	for _, mount := range m.Mounts {
		if mount.Source != nil {
			ensureNotEmpty()
			privileges.Mounts = append(privileges.Mounts, *mount.Source)
		}
	}
	for _, device := range m.Devices {
		ensureNotEmpty()
		privileges.Devices = append(privileges.Devices, *device.Path)
	}
	if len(m.Capabilities) > 0 {
		ensureNotEmpty()
		privileges.Capabilities = m.Capabilities
	}
	return
}
