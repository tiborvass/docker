package snapshot

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/snapshots"
	"github.com/tiborvass/docker/daemon/graphdriver"
	"github.com/tiborvass/docker/layer"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/snapshot"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var keyParent = []byte("parent")
var keyCommitted = []byte("committed")
var keyChainID = []byte("chainid")
var keySize = []byte("size")

type Opt struct {
	GraphDriver graphdriver.Driver
	LayerStore  layer.Store
	Root        string
}

type graphIDRegistrar interface {
	RegisterByGraphID(string, layer.ChainID, layer.DiffID, string, int64) (layer.Layer, error)
	Release(layer.Layer) ([]layer.Metadata, error)
	checksumCalculator
}

type checksumCalculator interface {
	ChecksumForGraphID(id, parent, oldTarDataPath, newTarDataPath string) (diffID layer.DiffID, size int64, err error)
}

type snapshotter struct {
	opt Opt

	refs map[string]layer.Layer
	db   *bolt.DB
	mu   sync.Mutex
	reg  graphIDRegistrar
}

var _ snapshot.SnapshotterBase = &snapshotter{}

func NewSnapshotter(opt Opt) (snapshot.SnapshotterBase, error) {
	dbPath := filepath.Join(opt.Root, "snapshots.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open database file %s", dbPath)
	}

	reg, ok := opt.LayerStore.(graphIDRegistrar)
	if !ok {
		return nil, errors.Errorf("layerstore doesn't support graphID registration")
	}

	s := &snapshotter{
		opt:  opt,
		db:   db,
		refs: map[string]layer.Layer{},
		reg:  reg,
	}
	return s, nil
}

func (s *snapshotter) Prepare(ctx context.Context, key, parent string, opts ...snapshots.Opt) error {
	origParent := parent
	if parent != "" {
		if l, err := s.getLayer(parent); err != nil {
			return err
		} else if l != nil {
			parent, err = getGraphID(l)
			if err != nil {
				return err
			}
		} else {
			parent, _ = s.getGraphDriverID(parent)
		}
	}
	if err := s.opt.GraphDriver.Create(key, parent, nil); err != nil {
		return err
	}
	if err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(key))
		if err != nil {
			return err
		}

		if err := b.Put(keyParent, []byte(origParent)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *snapshotter) chainID(key string) (layer.ChainID, bool) {
	if strings.HasPrefix(key, "sha256:") {
		dgst, err := digest.Parse(key)
		if err != nil {
			return "", false
		}
		return layer.ChainID(dgst), true
	}
	return "", false
}

func (s *snapshotter) getLayer(key string) (layer.Layer, error) {
	s.mu.Lock()
	l, ok := s.refs[key]
	if !ok {
		id, ok := s.chainID(key)
		if !ok {
			s.mu.Unlock()
			return nil, nil
		}
		var err error
		l, err = s.opt.LayerStore.Get(id)
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
		s.refs[string(id)] = l
		if err := s.db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(id))
			return err
		}); err != nil {
			s.mu.Unlock()
			return nil, err
		}
	}
	s.mu.Unlock()

	return l, nil
}

func (s *snapshotter) getGraphDriverID(key string) (string, bool) {
	var gdID string
	if err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(key))
		if b == nil {
			return errors.Errorf("not found") // TODO: typed
		}
		v := b.Get(keyCommitted)
		if v != nil {
			gdID = string(v)
		}
		return nil
	}); err != nil || gdID == "" {
		return key, false
	}
	return gdID, true
}

func (s *snapshotter) Stat(ctx context.Context, key string) (snapshots.Info, error) {
	if l, err := s.getLayer(key); err != nil {
		return snapshots.Info{}, err
	} else if l != nil {
		var parentID string
		if p := l.Parent(); p != nil {
			parentID = p.ChainID().String()
		}
		info := snapshots.Info{
			Kind:   snapshots.KindCommitted,
			Name:   key,
			Parent: parentID,
		}
		return info, nil
	}

	inf := snapshots.Info{
		Kind: snapshots.KindActive,
	}

	id, committed := s.getGraphDriverID(key)
	if committed {
		inf.Kind = snapshots.KindCommitted
	}

	if err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(id))
		if b == nil {
			return errors.Errorf("not found") // TODO: typed
		}
		inf.Name = string(key)
		v := b.Get(keyParent)
		if v != nil {
			inf.Parent = string(v)
		}
		return nil
	}); err != nil {
		return snapshots.Info{}, err
	}
	return inf, nil
}

func (s *snapshotter) Mounts(ctx context.Context, key string) (snapshot.MountFactory, error) {
	l, err := s.getLayer(key)
	if err != nil {
		return nil, err
	}
	if l != nil {
		id := identity.NewID()
		rwlayer, err := s.opt.LayerStore.CreateRWLayer(id, l.ChainID(), nil)
		if err != nil {
			return nil, err
		}
		rootfs, err := rwlayer.Mount("")
		if err != nil {
			return nil, err
		}
		mnt := []mount.Mount{{
			Source:  rootfs.Path(),
			Type:    "bind",
			Options: []string{"rbind"},
		}}
		return &constMountFactory{
			mounts: mnt,
			release: func() error {
				_, err := s.opt.LayerStore.ReleaseRWLayer(rwlayer)
				return err
			},
		}, nil
	}

	id, _ := s.getGraphDriverID(key)

	rootfs, err := s.opt.GraphDriver.Get(id, "")
	if err != nil {
		return nil, err
	}
	mnt := []mount.Mount{{
		Source:  rootfs.Path(),
		Type:    "bind",
		Options: []string{"rbind"},
	}}
	return &constMountFactory{
		mounts: mnt,
		release: func() error {
			return s.opt.GraphDriver.Put(id)
		},
	}, nil
}

func (s *snapshotter) Remove(ctx context.Context, key string) error {
	l, err := s.getLayer(key)
	if err != nil {
		return err
	}

	var found bool
	if err := s.db.Update(func(tx *bolt.Tx) error {
		found = tx.Bucket([]byte(key)) != nil
		if found {
			id, _ := s.getGraphDriverID(key)
			tx.DeleteBucket([]byte(key))
			if id != key {
				tx.DeleteBucket([]byte(id))
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if l != nil {
		_, err := s.opt.LayerStore.Release(l)
		return err
	}

	if !found { // this happens when removing views
		return nil
	}

	return s.opt.GraphDriver.Remove(key)
}

func (s *snapshotter) Commit(ctx context.Context, name, key string, opts ...snapshots.Opt) error {
	if err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return err
		}
		if err := b.Put(keyCommitted, []byte(key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	logrus.Debugf("committed %s as %s", name, key)
	return nil
}

func (s *snapshotter) View(ctx context.Context, key, parent string, opts ...snapshots.Opt) (snapshot.MountFactory, error) {
	return s.Mounts(ctx, parent)
}

func (s *snapshotter) Walk(ctx context.Context, fn func(context.Context, snapshots.Info) error) error {
	allKeys := map[string]struct{}{}
	commitedIDs := map[string]string{}
	chainIDs := map[string]layer.ChainID{}

	if err := s.db.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			allKeys[string(name)] = struct{}{}
			v := b.Get(keyCommitted)
			if v != nil {
				commitedIDs[string(v)] = string(name)
			}

			v = b.Get(keyChainID)
			if v != nil {
				chainIDs[string(name)] = layer.ChainID(v)
			}
			return nil
		})
		return nil
	}); err != nil {
		return err
	}

	for k := range allKeys {
		if _, ok := commitedIDs[k]; ok {
			continue
		}
		if chainID, ok := chainIDs[k]; ok {
			s.mu.Lock()
			if _, ok := s.refs[k]; !ok {
				l, err := s.opt.LayerStore.Get(chainID)
				if err != nil {
					s.mu.Unlock()
					return err
				}
				s.refs[k] = l
			}
			s.mu.Unlock()
		}

		if _, err := s.getLayer(k); err != nil {
			s.Remove(ctx, k)
			continue
		}
		info, err := s.Stat(ctx, k)
		if err != nil {
			s.Remove(ctx, k)
			continue
		}
		if err := fn(ctx, info); err != nil {
			return err
		}
	}

	return nil
}

func (s *snapshotter) Update(ctx context.Context, info snapshots.Info, fieldpaths ...string) (snapshots.Info, error) {
	// not implemented
	return s.Stat(ctx, info.Name)
}

func (s *snapshotter) Usage(ctx context.Context, key string) (snapshots.Usage, error) {
	return snapshots.Usage{}, nil
}

func (s *snapshotter) Close() error {
	return s.db.Close()
}

type constMountFactory struct {
	mounts  []mount.Mount
	release func() error
}

func (mf *constMountFactory) Mount() ([]mount.Mount, func() error, error) {
	release := mf.release
	if release == nil {
		release = func() error { return nil }
	}
	return mf.mounts, release, nil
}
