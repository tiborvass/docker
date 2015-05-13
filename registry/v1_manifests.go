package registry

import (
	"errors"

	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"
)

type manifests struct {
	*repository
	tagMap map[string]string
}

func (m *manifests) Exists(dgst digest.Digest) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *manifests) Get(dgst digest.Digest) (*manifest.SignedManifest, error) {
	return nil, errors.New("not implemented")
}

func (m *manifests) Delete(dgst digest.Digest) error {
	return errors.New("not implemented")
}

func (m *manifests) Put(manifest *manifest.SignedManifest) error {
	if m.currentEndpoint == "" {
		return errors.New("Internal endpoint not set, make sure you call BlobService.Stat or BlobService.Open before ManifestService.Put")
	}
	imgID := string(manifest.Raw)
	delete(m.jsonData, imgID)
	return m.session.PushRegistryTag(m.name, imgID, manifest.Tag, m.currentEndpoint)
}

func (m *manifests) ensureTags() (err error) {
	// only fetch tags if they were not already fetched for this repository session
	if m.tagMap == nil {
		if m.repoData == nil {
			m.repoData, err = m.session.GetRepositoryData(m.name)
			if err != nil {
				return err
			}
		}
		m.tagMap, err = m.session.GetRemoteTags(m.repoData.Endpoints, m.name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *manifests) Tags() ([]string, error) {
	if err := m.ensureTags(); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(m.tagMap))
	for tag := range m.tagMap {
		tags = append(tags, tag)
	}
	return tags, nil
}

func (m *manifests) ExistsByTag(tag string) (bool, error) {
	if err := m.ensureTags(); err != nil {
		return false, err
	}
	_, ok := m.tagMap[tag]
	return ok, nil
}

func (m *manifests) GetByTag(tag string) (*manifest.SignedManifest, error) {
	if err := m.ensureTags(); err != nil {
		return nil, err
	}
	id, ok := m.tagMap[tag]
	if !ok {
		return nil, nil
	}

	var (
		fslayers []manifest.FSLayer
		history  []manifest.History
	)
	if success, errs := m.loopEndpoints(true, func(endpoint string) error {
		dependentLayerIds, err := m.session.GetRemoteHistory(id, endpoint)
		if err != nil {
			return err
		}
		history = make([]manifest.History, len(dependentLayerIds))
		fslayers = make([]manifest.FSLayer, len(dependentLayerIds))
		for i, dependentLayerId := range dependentLayerIds {
			v1json, size, err := m.session.GetRemoteImageJSON(dependentLayerId, endpoint)
			if err != nil {
				return err
			}
			fslayers[i] = manifest.FSLayer{BlobSum: imgIDToDigest(dependentLayerId)}
			// hack: we're using the repository struct to pass the size information from
			// the manifests to the blobs service
			m.sizes[dependentLayerId] = int64(size)
			history[i] = manifest.History{V1Compatibility: string(v1json)}
		}
		return nil
	}); !success {
		return nil, errs
	}

	manifest := &manifest.SignedManifest{
		Manifest: manifest.Manifest{
			Versioned: manifest.Versioned{SchemaVersion: 1},
			Name:      tag,
			FSLayers:  fslayers,
			History:   history,
		},
	}
	return manifest, nil
}

func digestToImgID(dgst digest.Digest) string {
	// strip 'random:'
	return string(dgst[7:])
}

func imgIDToDigest(id string) digest.Digest {
	return digest.Digest("random:" + id)
}
