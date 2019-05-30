package containerimage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	distref "github.com/docker/distribution/reference"
	"github.com/docker/docker/layer"
	"github.com/moby/buildkit/exporter"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	keyImageName = "name"
)

// Differ can make a moby layer from a snapshot
type Differ interface {
	EnsureLayer(ctx context.Context, key string) ([]layer.DiffID, error)
}

// Opt defines a struct for creating new exporter
type Opt struct {
	ImageStore   images.Store
	ContentStore content.Store
	Differ       Differ
}

type imageExporter struct {
	opt Opt
}

// New creates a new moby imagestore exporter
func New(opt Opt) (exporter.Exporter, error) {
	im := &imageExporter{opt: opt}
	return im, nil
}

func (e *imageExporter) Resolve(ctx context.Context, opt map[string]string) (exporter.ExporterInstance, error) {
	i := &imageExporterInstance{imageExporter: e}
	for k, v := range opt {
		switch k {
		case keyImageName:
			for _, v := range strings.Split(v, ",") {
				ref, err := distref.ParseNormalizedNamed(v)
				if err != nil {
					return nil, err
				}
				i.targetNames = append(i.targetNames, ref)
			}
		case exptypes.ExporterImageConfigKey:
			if i.meta == nil {
				i.meta = make(map[string][]byte)
			}
			i.meta[k] = []byte(v)
		default:
			logrus.Warnf("image exporter: unknown option %s", k)
		}
	}
	return i, nil
}

type imageExporterInstance struct {
	*imageExporter
	targetNames []distref.Named
	meta        map[string][]byte
}

func (e *imageExporterInstance) Name() string {
	return "exporting to image"
}

func (e *imageExporterInstance) Export(ctx context.Context, inp exporter.Source) (map[string]string, error) {

	if len(inp.Refs) > 1 {
		return nil, fmt.Errorf("exporting multiple references to image store is currently unsupported")
	}

	ref := inp.Ref
	if ref != nil && len(inp.Refs) == 1 {
		return nil, fmt.Errorf("invalid exporter input: Ref and Refs are mutually exclusive")
	}

	// only one loop
	for _, v := range inp.Refs {
		ref = v
	}

	created := time.Now().UTC()

	var config []byte
	switch len(inp.Refs) {
	case 0:
		config = inp.Metadata[exptypes.ExporterImageConfigKey]
	case 1:
		platformsBytes, ok := inp.Metadata[exptypes.ExporterPlatformsKey]
		if !ok {
			return nil, fmt.Errorf("cannot export image, missing platforms mapping")
		}
		var p exptypes.Platforms
		if err := json.Unmarshal(platformsBytes, &p); err != nil {
			return nil, errors.Wrapf(err, "failed to parse platforms passed to exporter")
		}
		if len(p.Platforms) != len(inp.Refs) {
			return nil, errors.Errorf("number of platforms does not match references %d %d", len(p.Platforms), len(inp.Refs))
		}
		config = inp.Metadata[fmt.Sprintf("%s/%s", exptypes.ExporterImageConfigKey, p.Platforms[0].ID)]
	}

	var diffs []digest.Digest
	if ref != nil {
		layersDone := oneOffProgress(ctx, "exporting layers")

		if err := ref.Finalize(ctx, true); err != nil {
			return nil, err
		}

		diffIDs, err := e.opt.Differ.EnsureLayer(ctx, ref.ID())
		if err != nil {
			return nil, err
		}

		diffs = make([]digest.Digest, len(diffIDs))
		for i := range diffIDs {
			diffs[i] = digest.Digest(diffIDs[i])
		}

		layersDone(nil)
	}

	if len(config) == 0 {
		var err error
		config, err = emptyImageConfig()
		if err != nil {
			return nil, err
		}
	}

	history, err := parseHistoryFromConfig(config)
	if err != nil {
		return nil, err
	}

	diffs, history = normalizeLayersAndHistory(diffs, history, ref)

	config, err = patchImageConfig(config, diffs, history, inp.Metadata[exptypes.ExporterInlineCache])
	if err != nil {
		return nil, err
	}

	configDigest := digest.FromBytes(config)

	configDone := oneOffProgress(ctx, fmt.Sprintf("writing image %s", configDigest))

	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    configDigest,
		Size:      int64(len(config)),
	}

	imgref := fmt.Sprintf("config-%s-%s", desc.Digest.Algorithm().String(), desc.Digest.Encoded())
	if err := content.WriteBlob(ctx, e.opt.ContentStore, imgref, bytes.NewReader(config), desc); err != nil {
		return nil, configDone(err)
	}

	img, err := e.opt.ImageStore.Create(ctx, images.Image{
		Name:      desc.Digest.String(),
		Target:    desc,
		CreatedAt: created,
		UpdatedAt: created,
	})
	if err != nil && errors.Cause(err) != errdefs.ErrAlreadyExists {
		return nil, configDone(err)
	}
	configDone(nil)

	// TODO(containerd): use labels for reference store?
	/*
		if e.opt.ReferenceStore != nil {
			for _, targetName := range e.targetNames {
				tagDone := oneOffProgress(ctx, "naming to "+targetName.String())

				if err := e.opt.ReferenceStore.AddTag(targetName, digest.Digest(id), true); err != nil {
					return nil, tagDone(err)
				}
				tagDone(nil)
			}
		}
	*/

	return map[string]string{
		"containerimage.digest": img.Target.Digest.String(),
	}, nil
}
