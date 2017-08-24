package layer

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"sync"

	"github.com/docker/distribution"
	"github.com/tiborvass/docker/daemon/graphdriver"
	"github.com/tiborvass/docker/pkg/idtools"
	"github.com/tiborvass/docker/pkg/plugingetter"
	"github.com/tiborvass/docker/pkg/stringid"
	"github.com/tiborvass/docker/pkg/system"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"
)

// maxLayerDepth represents the maximum number of
// layers which can be chained together. 125 was
// chosen to account for the 127 max in some
// graphdrivers plus the 2 additional layers
// used to create a rwlayer.
const maxLayerDepth = 125

type layerStore struct {
	store       MetadataStore
	drivers     map[string]graphdriver.Driver
	useTarSplit map[string]bool

	layerMap map[ChainID]*roLayer
	layerL   sync.Mutex

	mounts map[string]*mountedLayer
	mountL sync.Mutex
}

// StoreOptions are the options used to create a new Store instance
type StoreOptions struct {
	Root                      string
	GraphDrivers              map[string]string
	MetadataStorePathTemplate string
	GraphDriverOptions        []string
	IDMappings                *idtools.IDMappings
	PluginGetter              plugingetter.PluginGetter
	ExperimentalEnabled       bool
}

// NewStoreFromOptions creates a new Store instance
func NewStoreFromOptions(options StoreOptions) (Store, error) {
	drivers := make(map[string]graphdriver.Driver)
	for os, drivername := range options.GraphDrivers {
		var err error
		drivers[os], err = graphdriver.New(drivername,
			options.PluginGetter,
			graphdriver.Options{
				Root:                options.Root,
				DriverOptions:       options.GraphDriverOptions,
				UIDMaps:             options.IDMappings.UIDs(),
				GIDMaps:             options.IDMappings.GIDs(),
				ExperimentalEnabled: options.ExperimentalEnabled,
			})
		if err != nil {
			return nil, fmt.Errorf("error initializing graphdriver: %v", err)
		}
		logrus.Debugf("Initialized graph driver %s", drivername)
	}

	fms, err := NewFSMetadataStore(fmt.Sprintf(options.MetadataStorePathTemplate, options.GraphDrivers[runtime.GOOS]))
	if err != nil {
		return nil, err
	}

	return NewStoreFromGraphDrivers(fms, drivers)
}

// NewStoreFromGraphDrivers creates a new Store instance using the provided
// metadata store and graph drivers. The metadata store will be used to restore
// the Store.
func NewStoreFromGraphDrivers(store MetadataStore, drivers map[string]graphdriver.Driver) (Store, error) {

	useTarSplit := make(map[string]bool)
	for os, driver := range drivers {

		caps := graphdriver.Capabilities{}
		if capDriver, ok := driver.(graphdriver.CapabilityDriver); ok {
			caps = capDriver.Capabilities()
		}
		useTarSplit[os] = !caps.ReproducesExactDiffs
	}

	ls := &layerStore{
		store:       store,
		drivers:     drivers,
		layerMap:    map[ChainID]*roLayer{},
		mounts:      map[string]*mountedLayer{},
		useTarSplit: useTarSplit,
	}

	ids, mounts, err := store.List()
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		l, err := ls.loadLayer(id)
		if err != nil {
			logrus.Debugf("Failed to load layer %s: %s", id, err)
			continue
		}
		if l.parent != nil {
			l.parent.referenceCount++
		}
	}

	for _, mount := range mounts {
		if err := ls.loadMount(mount); err != nil {
			logrus.Debugf("Failed to load mount %s: %s", mount, err)
		}
	}

	return ls, nil
}

func (ls *layerStore) loadLayer(layer ChainID) (*roLayer, error) {
	cl, ok := ls.layerMap[layer]
	if ok {
		return cl, nil
	}

	diff, err := ls.store.GetDiffID(layer)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff id for %s: %s", layer, err)
	}

	size, err := ls.store.GetSize(layer)
	if err != nil {
		return nil, fmt.Errorf("failed to get size for %s: %s", layer, err)
	}

	cacheID, err := ls.store.GetCacheID(layer)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache id for %s: %s", layer, err)
	}

	parent, err := ls.store.GetParent(layer)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent for %s: %s", layer, err)
	}

	descriptor, err := ls.store.GetDescriptor(layer)
	if err != nil {
		return nil, fmt.Errorf("failed to get descriptor for %s: %s", layer, err)
	}

	os, err := ls.store.GetOS(layer)
	if err != nil {
		return nil, fmt.Errorf("failed to get operating system for %s: %s", layer, err)
	}

	cl = &roLayer{
		chainID:    layer,
		diffID:     diff,
		size:       size,
		cacheID:    cacheID,
		layerStore: ls,
		references: map[Layer]struct{}{},
		descriptor: descriptor,
		os:         os,
	}

	if parent != "" {
		p, err := ls.loadLayer(parent)
		if err != nil {
			return nil, err
		}
		cl.parent = p
	}

	ls.layerMap[cl.chainID] = cl

	return cl, nil
}

func (ls *layerStore) loadMount(mount string) error {
	if _, ok := ls.mounts[mount]; ok {
		return nil
	}

	mountID, err := ls.store.GetMountID(mount)
	if err != nil {
		return err
	}

	initID, err := ls.store.GetInitID(mount)
	if err != nil {
		return err
	}

	parent, err := ls.store.GetMountParent(mount)
	if err != nil {
		return err
	}

	ml := &mountedLayer{
		name:       mount,
		mountID:    mountID,
		initID:     initID,
		layerStore: ls,
		references: map[RWLayer]*referencedRWLayer{},
	}

	if parent != "" {
		p, err := ls.loadLayer(parent)
		if err != nil {
			return err
		}
		ml.parent = p

		p.referenceCount++
	}

	ls.mounts[ml.name] = ml

	return nil
}

func (ls *layerStore) applyTar(tx MetadataTransaction, ts io.Reader, parent string, layer *roLayer) error {
	digester := digest.Canonical.Digester()
	tr := io.TeeReader(ts, digester.Hash())

	rdr := tr
	if ls.useTarSplit[layer.os] {
		tsw, err := tx.TarSplitWriter(true)
		if err != nil {
			return err
		}
		metaPacker := storage.NewJSONPacker(tsw)
		defer tsw.Close()

		// we're passing nil here for the file putter, because the ApplyDiff will
		// handle the extraction of the archive
		rdr, err = asm.NewInputTarStream(tr, metaPacker, nil)
		if err != nil {
			return err
		}
	}

	applySize, err := ls.drivers[layer.os].ApplyDiff(layer.cacheID, parent, rdr)
	if err != nil {
		return err
	}

	// Discard trailing data but ensure metadata is picked up to reconstruct stream
	io.Copy(ioutil.Discard, rdr) // ignore error as reader may be closed

	layer.size = applySize
	layer.diffID = DiffID(digester.Digest())

	logrus.Debugf("Applied tar %s to %s, size: %d", layer.diffID, layer.cacheID, applySize)

	return nil
}

func (ls *layerStore) Register(ts io.Reader, parent ChainID, os string) (Layer, error) {
	return ls.registerWithDescriptor(ts, parent, os, distribution.Descriptor{})
}

func (ls *layerStore) registerWithDescriptor(ts io.Reader, parent ChainID, os string, descriptor distribution.Descriptor) (Layer, error) {
	// err is used to hold the error which will always trigger
	// cleanup of creates sources but may not be an error returned
	// to the caller (already exists).
	var err error
	var pid string
	var p *roLayer

	if string(parent) != "" {
		p = ls.get(parent)
		if p == nil {
			return nil, ErrLayerDoesNotExist
		}
		pid = p.cacheID
		// Release parent chain if error
		defer func() {
			if err != nil {
				ls.layerL.Lock()
				ls.releaseLayer(p)
				ls.layerL.Unlock()
			}
		}()
		if p.depth() >= maxLayerDepth {
			err = ErrMaxDepthExceeded
			return nil, err
		}
	}

	// Validate the operating system is valid
	if os == "" {
		os = runtime.GOOS
	}
	if err := system.ValidatePlatform(system.ParsePlatform(os)); err != nil {
		return nil, err
	}

	// Create new roLayer
	layer := &roLayer{
		parent:         p,
		cacheID:        stringid.GenerateRandomID(),
		referenceCount: 1,
		layerStore:     ls,
		references:     map[Layer]struct{}{},
		descriptor:     descriptor,
		os:             os,
	}

	if err = ls.drivers[os].Create(layer.cacheID, pid, nil); err != nil {
		return nil, err
	}

	tx, err := ls.store.StartTransaction()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			logrus.Debugf("Cleaning up layer %s: %v", layer.cacheID, err)
			if err := ls.drivers[os].Remove(layer.cacheID); err != nil {
				logrus.Errorf("Error cleaning up cache layer %s: %v", layer.cacheID, err)
			}
			if err := tx.Cancel(); err != nil {
				logrus.Errorf("Error canceling metadata transaction %q: %s", tx.String(), err)
			}
		}
	}()

	if err = ls.applyTar(tx, ts, pid, layer); err != nil {
		return nil, err
	}

	if layer.parent == nil {
		layer.chainID = ChainID(layer.diffID)
	} else {
		layer.chainID = createChainIDFromParent(layer.parent.chainID, layer.diffID)
	}

	if err = storeLayer(tx, layer); err != nil {
		return nil, err
	}

	ls.layerL.Lock()
	defer ls.layerL.Unlock()

	if existingLayer := ls.getWithoutLock(layer.chainID); existingLayer != nil {
		// Set error for cleanup, but do not return the error
		err = errors.New("layer already exists")
		return existingLayer.getReference(), nil
	}

	if err = tx.Commit(layer.chainID); err != nil {
		return nil, err
	}

	ls.layerMap[layer.chainID] = layer

	return layer.getReference(), nil
}

func (ls *layerStore) getWithoutLock(layer ChainID) *roLayer {
	l, ok := ls.layerMap[layer]
	if !ok {
		return nil
	}

	l.referenceCount++

	return l
}

func (ls *layerStore) get(l ChainID) *roLayer {
	ls.layerL.Lock()
	defer ls.layerL.Unlock()
	return ls.getWithoutLock(l)
}

func (ls *layerStore) Get(l ChainID) (Layer, error) {
	ls.layerL.Lock()
	defer ls.layerL.Unlock()

	layer := ls.getWithoutLock(l)
	if layer == nil {
		return nil, ErrLayerDoesNotExist
	}

	return layer.getReference(), nil
}

func (ls *layerStore) Map() map[ChainID]Layer {
	ls.layerL.Lock()
	defer ls.layerL.Unlock()

	layers := map[ChainID]Layer{}

	for k, v := range ls.layerMap {
		layers[k] = v
	}

	return layers
}

func (ls *layerStore) deleteLayer(layer *roLayer, metadata *Metadata) error {
	err := ls.drivers[layer.os].Remove(layer.cacheID)
	if err != nil {
		return err
	}
	err = ls.store.Remove(layer.chainID)
	if err != nil {
		return err
	}
	metadata.DiffID = layer.diffID
	metadata.ChainID = layer.chainID
	metadata.Size, err = layer.Size()
	if err != nil {
		return err
	}
	metadata.DiffSize = layer.size

	return nil
}

func (ls *layerStore) releaseLayer(l *roLayer) ([]Metadata, error) {
	depth := 0
	removed := []Metadata{}
	for {
		if l.referenceCount == 0 {
			panic("layer not retained")
		}
		l.referenceCount--
		if l.referenceCount != 0 {
			return removed, nil
		}

		if len(removed) == 0 && depth > 0 {
			panic("cannot remove layer with child")
		}
		if l.hasReferences() {
			panic("cannot delete referenced layer")
		}
		var metadata Metadata
		if err := ls.deleteLayer(l, &metadata); err != nil {
			return nil, err
		}

		delete(ls.layerMap, l.chainID)
		removed = append(removed, metadata)

		if l.parent == nil {
			return removed, nil
		}

		depth++
		l = l.parent
	}
}

func (ls *layerStore) Release(l Layer) ([]Metadata, error) {
	ls.layerL.Lock()
	defer ls.layerL.Unlock()
	layer, ok := ls.layerMap[l.ChainID()]
	if !ok {
		return []Metadata{}, nil
	}
	if !layer.hasReference(l) {
		return nil, ErrLayerNotRetained
	}

	layer.deleteReference(l)

	return ls.releaseLayer(layer)
}

func (ls *layerStore) CreateRWLayer(name string, parent ChainID, os string, opts *CreateRWLayerOpts) (RWLayer, error) {
	var (
		storageOpt map[string]string
		initFunc   MountInit
		mountLabel string
	)

	if opts != nil {
		mountLabel = opts.MountLabel
		storageOpt = opts.StorageOpt
		initFunc = opts.InitFunc
	}

	ls.mountL.Lock()
	defer ls.mountL.Unlock()
	m, ok := ls.mounts[name]
	if ok {
		return nil, ErrMountNameConflict
	}

	var err error
	var pid string
	var p *roLayer
	if string(parent) != "" {
		p = ls.get(parent)
		if p == nil {
			return nil, ErrLayerDoesNotExist
		}
		pid = p.cacheID

		// Release parent chain if error
		defer func() {
			if err != nil {
				ls.layerL.Lock()
				ls.releaseLayer(p)
				ls.layerL.Unlock()
			}
		}()
	}

	// Ensure the operating system is set to the host OS if not populated.
	if os == "" {
		os = runtime.GOOS
	}
	m = &mountedLayer{
		name:       name,
		parent:     p,
		mountID:    ls.mountID(name),
		layerStore: ls,
		references: map[RWLayer]*referencedRWLayer{},
		os:         os,
	}

	if initFunc != nil {
		pid, err = ls.initMount(m.mountID, m.os, pid, mountLabel, initFunc, storageOpt)
		if err != nil {
			return nil, err
		}
		m.initID = pid
	}

	createOpts := &graphdriver.CreateOpts{
		StorageOpt: storageOpt,
	}

	if err = ls.drivers[os].CreateReadWrite(m.mountID, pid, createOpts); err != nil {
		return nil, err
	}
	if err = ls.saveMount(m); err != nil {
		return nil, err
	}

	return m.getReference(), nil
}

func (ls *layerStore) GetRWLayer(id string) (RWLayer, error) {
	ls.mountL.Lock()
	defer ls.mountL.Unlock()
	mount, ok := ls.mounts[id]
	if !ok {
		return nil, ErrMountDoesNotExist
	}

	return mount.getReference(), nil
}

func (ls *layerStore) GetMountID(id string) (string, error) {
	ls.mountL.Lock()
	defer ls.mountL.Unlock()
	mount, ok := ls.mounts[id]
	if !ok {
		return "", ErrMountDoesNotExist
	}
	logrus.Debugf("GetMountID id: %s -> mountID: %s", id, mount.mountID)

	return mount.mountID, nil
}

func (ls *layerStore) ReleaseRWLayer(l RWLayer) ([]Metadata, error) {
	ls.mountL.Lock()
	defer ls.mountL.Unlock()
	m, ok := ls.mounts[l.Name()]
	if !ok {
		return []Metadata{}, nil
	}

	if err := m.deleteReference(l); err != nil {
		return nil, err
	}

	if m.hasReferences() {
		return []Metadata{}, nil
	}

	if err := ls.drivers[l.OS()].Remove(m.mountID); err != nil {
		logrus.Errorf("Error removing mounted layer %s: %s", m.name, err)
		m.retakeReference(l)
		return nil, err
	}

	if m.initID != "" {
		if err := ls.drivers[l.OS()].Remove(m.initID); err != nil {
			logrus.Errorf("Error removing init layer %s: %s", m.name, err)
			m.retakeReference(l)
			return nil, err
		}
	}

	if err := ls.store.RemoveMount(m.name); err != nil {
		logrus.Errorf("Error removing mount metadata: %s: %s", m.name, err)
		m.retakeReference(l)
		return nil, err
	}

	delete(ls.mounts, m.Name())

	ls.layerL.Lock()
	defer ls.layerL.Unlock()
	if m.parent != nil {
		return ls.releaseLayer(m.parent)
	}

	return []Metadata{}, nil
}

func (ls *layerStore) saveMount(mount *mountedLayer) error {
	if err := ls.store.SetMountID(mount.name, mount.mountID); err != nil {
		return err
	}

	if mount.initID != "" {
		if err := ls.store.SetInitID(mount.name, mount.initID); err != nil {
			return err
		}
	}

	if mount.parent != nil {
		if err := ls.store.SetMountParent(mount.name, mount.parent.chainID); err != nil {
			return err
		}
	}

	ls.mounts[mount.name] = mount

	return nil
}

func (ls *layerStore) initMount(graphID, os, parent, mountLabel string, initFunc MountInit, storageOpt map[string]string) (string, error) {
	// Use "<graph-id>-init" to maintain compatibility with graph drivers
	// which are expecting this layer with this special name. If all
	// graph drivers can be updated to not rely on knowing about this layer
	// then the initID should be randomly generated.
	initID := fmt.Sprintf("%s-init", graphID)

	createOpts := &graphdriver.CreateOpts{
		MountLabel: mountLabel,
		StorageOpt: storageOpt,
	}

	if err := ls.drivers[os].CreateReadWrite(initID, parent, createOpts); err != nil {
		return "", err
	}
	p, err := ls.drivers[os].Get(initID, "")
	if err != nil {
		return "", err
	}

	if err := initFunc(p); err != nil {
		ls.drivers[os].Put(initID)
		return "", err
	}

	if err := ls.drivers[os].Put(initID); err != nil {
		return "", err
	}

	return initID, nil
}

func (ls *layerStore) getTarStream(rl *roLayer) (io.ReadCloser, error) {
	if !ls.useTarSplit[rl.os] {
		var parentCacheID string
		if rl.parent != nil {
			parentCacheID = rl.parent.cacheID
		}

		return ls.drivers[rl.os].Diff(rl.cacheID, parentCacheID)
	}

	r, err := ls.store.TarSplitReader(rl.chainID)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		err := ls.assembleTarTo(rl.cacheID, rl.os, r, nil, pw)
		if err != nil {
			pw.CloseWithError(err)
		} else {
			pw.Close()
		}
	}()

	return pr, nil
}

func (ls *layerStore) assembleTarTo(graphID, os string, metadata io.ReadCloser, size *int64, w io.Writer) error {
	diffDriver, ok := ls.drivers[os].(graphdriver.DiffGetterDriver)
	if !ok {
		diffDriver = &naiveDiffPathDriver{ls.drivers[os]}
	}

	defer metadata.Close()

	// get our relative path to the container
	fileGetCloser, err := diffDriver.DiffGetter(graphID)
	if err != nil {
		return err
	}
	defer fileGetCloser.Close()

	metaUnpacker := storage.NewJSONUnpacker(metadata)
	upackerCounter := &unpackSizeCounter{metaUnpacker, size}
	logrus.Debugf("Assembling tar data for %s", graphID)
	return asm.WriteOutputTarStream(fileGetCloser, upackerCounter, w)
}

func (ls *layerStore) Cleanup() error {
	var err error
	for _, driver := range ls.drivers {
		if e := driver.Cleanup(); e != nil {
			err = fmt.Errorf("%s - %s", err.Error(), e.Error())
		}
	}
	return err
}

func (ls *layerStore) DriverStatus(os string) [][2]string {
	if os == "" {
		os = runtime.GOOS
	}
	return ls.drivers[os].Status()
}

func (ls *layerStore) DriverName(os string) string {
	if os == "" {
		os = runtime.GOOS
	}
	return ls.drivers[os].String()
}

type naiveDiffPathDriver struct {
	graphdriver.Driver
}

type fileGetPutter struct {
	storage.FileGetter
	driver graphdriver.Driver
	id     string
}

func (w *fileGetPutter) Close() error {
	return w.driver.Put(w.id)
}

func (n *naiveDiffPathDriver) DiffGetter(id string) (graphdriver.FileGetCloser, error) {
	p, err := n.Driver.Get(id, "")
	if err != nil {
		return nil, err
	}
	return &fileGetPutter{storage.NewPathFileGetter(p.Path()), n.Driver, id}, nil
}
