package graph

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/engine"
	"github.com/tiborvass/docker/pkg/archive"
	"github.com/tiborvass/docker/pkg/parsers"
	"github.com/tiborvass/docker/registry"
)

// CmdImageExport exports all images with the given tag. All versions
// containing the same tag are exported. The resulting output is an
// uncompressed tar ball.
// name is the set of tags to export.
// out is the writer where the images are written to.
type ImageExportConfig struct {
	Names     []string
	Outstream io.Writer
	Engine    *engine.Engine
}

func (s *TagStore) ImageExport(imageExportConfig *ImageExportConfig) error {

	// get image json
	tempdir, err := ioutil.TempDir("", "docker-export-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempdir)

	rootRepoMap := map[string]Repository{}
	addKey := func(name string, tag string, id string) {
		logrus.Debugf("add key [%s:%s]", name, tag)
		if repo, ok := rootRepoMap[name]; !ok {
			rootRepoMap[name] = Repository{tag: id}
		} else {
			repo[tag] = id
		}
	}
	for _, name := range imageExportConfig.Names {
		name = registry.NormalizeLocalName(name)
		logrus.Debugf("Serializing %s", name)
		rootRepo := s.Repositories[name]
		if rootRepo != nil {
			// this is a base repo name, like 'busybox'
			for tag, id := range rootRepo {
				addKey(name, tag, id)
				if err := s.exportImage(imageExportConfig.Engine, id, tempdir); err != nil {
					return err
				}
			}
		} else {
			img, err := s.LookupImage(name)
			if err != nil {
				return err
			}

			if img != nil {
				// This is a named image like 'busybox:latest'
				repoName, repoTag := parsers.ParseRepositoryTag(name)

				// check this length, because a lookup of a truncated has will not have a tag
				// and will not need to be added to this map
				if len(repoTag) > 0 {
					addKey(repoName, repoTag, img.ID)
				}
				if err := s.exportImage(imageExportConfig.Engine, img.ID, tempdir); err != nil {
					return err
				}

			} else {
				// this must be an ID that didn't get looked up just right?
				if err := s.exportImage(imageExportConfig.Engine, name, tempdir); err != nil {
					return err
				}
			}
		}
		logrus.Debugf("End Serializing %s", name)
	}
	// write repositories, if there is something to write
	if len(rootRepoMap) > 0 {
		rootRepoJson, _ := json.Marshal(rootRepoMap)
		if err := ioutil.WriteFile(path.Join(tempdir, "repositories"), rootRepoJson, os.FileMode(0644)); err != nil {
			return err
		}
	} else {
		logrus.Debugf("There were no repositories to write")
	}

	fs, err := archive.Tar(tempdir, archive.Uncompressed)
	if err != nil {
		return err
	}
	defer fs.Close()

	if _, err := io.Copy(imageExportConfig.Outstream, fs); err != nil {
		return err
	}
	logrus.Debugf("End export image")
	return nil
}

// FIXME: this should be a top-level function, not a class method
func (s *TagStore) exportImage(eng *engine.Engine, name, tempdir string) error {
	for n := name; n != ""; {
		// temporary directory
		tmpImageDir := path.Join(tempdir, n)
		if err := os.Mkdir(tmpImageDir, os.FileMode(0755)); err != nil {
			if os.IsExist(err) {
				return nil
			}
			return err
		}

		var version = "1.0"
		var versionBuf = []byte(version)

		if err := ioutil.WriteFile(path.Join(tmpImageDir, "VERSION"), versionBuf, os.FileMode(0644)); err != nil {
			return err
		}

		// serialize json
		json, err := os.Create(path.Join(tmpImageDir, "json"))
		if err != nil {
			return err
		}
		job := eng.Job("image_inspect", n)
		job.SetenvBool("raw", true)
		job.Stdout.Add(json)
		if err := job.Run(); err != nil {
			return err
		}

		// serialize filesystem
		fsTar, err := os.Create(path.Join(tmpImageDir, "layer.tar"))
		if err != nil {
			return err
		}
		if err := s.ImageTarLayer(n, fsTar); err != nil {
			return err
		}

		// find parent
		img, err := s.LookupImage(n)
		if err != nil {
			return err
		}
		n = img.Parent
	}
	return nil
}
