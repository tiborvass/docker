package graph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/engine"
	"github.com/tiborvass/docker/pkg/tarsum"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/runconfig"
	"github.com/docker/libtrust"
)

func (s *TagStore) CmdManifest(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("usage: %s NAME", job.Name)
	}
	name := job.Args[0]
	tag := job.Getenv("tag")
	if tag == "" {
		tag = "latest"
	}

	// Resolve the Repository name from fqn to endpoint + name
	repoInfo, err := registry.ParseRepositoryInfo(name)
	if err != nil {
		return job.Error(err)
	}

	manifestBytes, err := s.newManifest(name, repoInfo.RemoteName, tag)
	if err != nil {
		return job.Error(err)
	}

	_, err = job.Stdout.Write(manifestBytes)
	if err != nil {
		return job.Error(err)
	}

	return engine.StatusOK
}

func (s *TagStore) newManifest(localName, remoteName, tag string) ([]byte, error) {
	manifest := &registry.ManifestData{
		Name:          remoteName,
		Tag:           tag,
		SchemaVersion: 1,
	}
	localRepo, err := s.Get(localName)
	if err != nil {
		return nil, err
	}
	if localRepo == nil {
		return nil, fmt.Errorf("Repo does not exist: %s", localName)
	}

	// Get the top-most layer id which the tag points to
	layerId, exists := localRepo[tag]
	if !exists {
		return nil, fmt.Errorf("Tag does not exist for %s: %s", localName, tag)
	}
	layersSeen := make(map[string]bool)

	layer, err := s.graph.Get(layerId)
	if err != nil {
		return nil, err
	}
	if layer.Config == nil {
		return nil, errors.New("Missing layer configuration")
	}
	manifest.Architecture = layer.Architecture
	manifest.FSLayers = make([]*registry.FSLayer, 0, 4)
	manifest.History = make([]*registry.ManifestHistory, 0, 4)
	var metadata runconfig.Config
	metadata = *layer.Config

	for ; layer != nil; layer, err = layer.GetParent() {
		if err != nil {
			return nil, err
		}

		if layersSeen[layer.ID] {
			break
		}
		if layer.Config != nil && metadata.Image != layer.ID {
			err = runconfig.Merge(&metadata, layer.Config)
			if err != nil {
				return nil, err
			}
		}

		archive, err := layer.TarLayer()
		if err != nil {
			return nil, err
		}

		tarSum, err := tarsum.NewTarSum(archive, true, tarsum.Version1)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(ioutil.Discard, tarSum); err != nil {
			return nil, err
		}

		tarId := tarSum.Sum(nil)

		manifest.FSLayers = append(manifest.FSLayers, &registry.FSLayer{BlobSum: tarId})

		layersSeen[layer.ID] = true
		jsonData, err := ioutil.ReadFile(path.Join(s.graph.Root, layer.ID, "json"))
		if err != nil {
			return nil, fmt.Errorf("Cannot retrieve the path for {%s}: %s", layer.ID, err)
		}
		manifest.History = append(manifest.History, &registry.ManifestHistory{V1Compatibility: string(jsonData)})
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "   ")
	if err != nil {
		return nil, err
	}

	return manifestBytes, nil
}

func (s *TagStore) verifyManifest(eng *engine.Engine, manifestBytes []byte) (*registry.ManifestData, bool, error) {
	sig, err := libtrust.ParsePrettySignature(manifestBytes, "signatures")
	if err != nil {
		return nil, false, fmt.Errorf("error parsing payload: %s", err)
	}

	keys, err := sig.Verify()
	if err != nil {
		return nil, false, fmt.Errorf("error verifying payload: %s", err)
	}

	payload, err := sig.Payload()
	if err != nil {
		return nil, false, fmt.Errorf("error retrieving payload: %s", err)
	}

	var manifest registry.ManifestData
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return nil, false, fmt.Errorf("error unmarshalling manifest: %s", err)
	}
	if manifest.SchemaVersion != 1 {
		return nil, false, fmt.Errorf("unsupported schema version: %d", manifest.SchemaVersion)
	}

	var verified bool
	for _, key := range keys {
		job := eng.Job("trust_key_check")
		b, err := key.MarshalJSON()
		if err != nil {
			return nil, false, fmt.Errorf("error marshalling public key: %s", err)
		}
		namespace := manifest.Name
		if namespace[0] != '/' {
			namespace = "/" + namespace
		}
		stdoutBuffer := bytes.NewBuffer(nil)

		job.Args = append(job.Args, namespace)
		job.Setenv("PublicKey", string(b))
		// Check key has read/write permission (0x03)
		job.SetenvInt("Permission", 0x03)
		job.Stdout.Add(stdoutBuffer)
		if err = job.Run(); err != nil {
			return nil, false, fmt.Errorf("error running key check: %s", err)
		}
		result := engine.Tail(stdoutBuffer, 1)
		log.Debugf("Key check result: %q", result)
		if result == "verified" {
			verified = true
		}
	}

	return &manifest, verified, nil
}

func checkValidManifest(manifest *registry.ManifestData) error {
	if len(manifest.FSLayers) != len(manifest.History) {
		return fmt.Errorf("length of history not equal to number of layers")
	}

	if len(manifest.FSLayers) == 0 {
		return fmt.Errorf("no FSLayers in manifest")
	}

	return nil
}
