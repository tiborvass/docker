package daemon

import (
	"fmt"
	"sync/atomic"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/filters"
	"github.com/tiborvass/docker/layer"
	"github.com/tiborvass/docker/pkg/directory"
	"github.com/tiborvass/docker/volume"
	"github.com/opencontainers/go-digest"
)

func (daemon *Daemon) getLayerRefs() map[layer.ChainID]int {
	tmpImages := daemon.imageStore.Map()
	layerRefs := map[layer.ChainID]int{}
	for id, img := range tmpImages {
		dgst := digest.Digest(id)
		if len(daemon.referenceStore.References(dgst)) == 0 && len(daemon.imageStore.Children(id)) != 0 {
			continue
		}

		rootFS := *img.RootFS
		rootFS.DiffIDs = nil
		for _, id := range img.RootFS.DiffIDs {
			rootFS.Append(id)
			chid := rootFS.ChainID()
			layerRefs[chid]++
		}
	}

	return layerRefs
}

// SystemDiskUsage returns information about the daemon data disk usage
func (daemon *Daemon) SystemDiskUsage(ctx context.Context) (*types.DiskUsage, error) {
	if !atomic.CompareAndSwapInt32(&daemon.diskUsageRunning, 0, 1) {
		return nil, fmt.Errorf("a disk usage operation is already running")
	}
	defer atomic.StoreInt32(&daemon.diskUsageRunning, 0)

	// Retrieve container list
	allContainers, err := daemon.Containers(&types.ContainerListOptions{
		Size: true,
		All:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve container list: %v", err)
	}

	// Get all top images with extra attributes
	allImages, err := daemon.Images(filters.NewArgs(), false, true)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve image list: %v", err)
	}

	// Get all local volumes
	allVolumes := []*types.Volume{}
	getLocalVols := func(v volume.Volume) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			name := v.Name()
			refs := daemon.volumes.Refs(v)

			tv := volumeToAPIType(v)
			sz, err := directory.Size(v.Path())
			if err != nil {
				logrus.Warnf("failed to determine size of volume %v", name)
				sz = -1
			}
			tv.UsageData = &types.VolumeUsageData{Size: sz, RefCount: int64(len(refs))}
			allVolumes = append(allVolumes, tv)
		}

		return nil
	}

	err = daemon.traverseLocalVolumes(getLocalVols)
	if err != nil {
		return nil, err
	}

	// Get total layers size on disk
	layerRefs := daemon.getLayerRefs()
	allLayers := daemon.layerStore.Map()
	var allLayersSize int64
	for _, l := range allLayers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			size, err := l.DiffSize()
			if err == nil {
				if _, ok := layerRefs[l.ChainID()]; ok {
					allLayersSize += size
				} else {
					logrus.Warnf("found leaked image layer %v", l.ChainID())
				}
			} else {
				logrus.Warnf("failed to get diff size for layer %v", l.ChainID())
			}
		}
	}

	return &types.DiskUsage{
		LayersSize: allLayersSize,
		Containers: allContainers,
		Volumes:    allVolumes,
		Images:     allImages,
	}, nil
}
