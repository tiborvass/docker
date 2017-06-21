package daemon

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/builder/dockerfile"
	"github.com/tiborvass/docker/builder/remotecontext"
	"github.com/tiborvass/docker/dockerversion"
	"github.com/tiborvass/docker/image"
	"github.com/tiborvass/docker/layer"
	"github.com/tiborvass/docker/pkg/archive"
	"github.com/tiborvass/docker/pkg/progress"
	"github.com/tiborvass/docker/pkg/streamformatter"
	"github.com/pkg/errors"
)

// ImportImage imports an image, getting the archived layer data either from
// inConfig (if src is "-"), or from a URI specified in src. Progress output is
// written to outStream. Repository and tag names can optionally be given in
// the repo and tag arguments, respectively.
func (daemon *Daemon) ImportImage(src string, repository, tag string, msg string, inConfig io.ReadCloser, outStream io.Writer, changes []string) error {
	var (
		rc     io.ReadCloser
		resp   *http.Response
		newRef reference.Named
	)

	if repository != "" {
		var err error
		newRef, err = reference.ParseNormalizedNamed(repository)
		if err != nil {
			return err
		}
		if _, isCanonical := newRef.(reference.Canonical); isCanonical {
			return errors.New("cannot import digest reference")
		}

		if tag != "" {
			newRef, err = reference.WithTag(newRef, tag)
			if err != nil {
				return err
			}
		}
	}

	config, err := dockerfile.BuildFromConfig(&container.Config{}, changes)
	if err != nil {
		return err
	}
	if src == "-" {
		rc = inConfig
	} else {
		inConfig.Close()
		if len(strings.Split(src, "://")) == 1 {
			src = "http://" + src
		}
		u, err := url.Parse(src)
		if err != nil {
			return err
		}

		resp, err = remotecontext.GetWithStatusError(u.String())
		if err != nil {
			return err
		}
		outStream.Write(streamformatter.FormatStatus("", "Downloading from %s", u))
		progressOutput := streamformatter.NewJSONProgressOutput(outStream, true)
		rc = progress.NewProgressReader(resp.Body, progressOutput, resp.ContentLength, "", "Importing")
	}

	defer rc.Close()
	if len(msg) == 0 {
		msg = "Imported from " + src
	}

	inflatedLayerData, err := archive.DecompressStream(rc)
	if err != nil {
		return err
	}
	// TODO: support windows baselayer?
	// TODO: LCOW support @jhowardmsft. For now, pass in a null platform when
	//       registering the layer. Windows doesn't currently support import,
	//       but for Linux images, there's no reason it couldn't. However it
	//       would need another CLI flag as there's no meta-data indicating
	//       the OS of the thing being imported.
	l, err := daemon.stores[runtime.GOOS].layerStore.Register(inflatedLayerData, "", "")
	if err != nil {
		return err
	}
	defer layer.ReleaseAndLog(daemon.stores[runtime.GOOS].layerStore, l) // TODO LCOW @jhowardmsft as for above comment

	created := time.Now().UTC()
	imgConfig, err := json.Marshal(&image.Image{
		V1Image: image.V1Image{
			DockerVersion: dockerversion.Version,
			Config:        config,
			Architecture:  runtime.GOARCH,
			OS:            runtime.GOOS, // TODO LCOW @jhowardmsft as for above commment
			Created:       created,
			Comment:       msg,
		},
		RootFS: &image.RootFS{
			Type:    "layers",
			DiffIDs: []layer.DiffID{l.DiffID()},
		},
		History: []image.History{{
			Created: created,
			Comment: msg,
		}},
	})
	if err != nil {
		return err
	}

	// TODO @jhowardmsft LCOW - Again, assume the OS of the host for now
	id, err := daemon.stores[runtime.GOOS].imageStore.Create(imgConfig)
	if err != nil {
		return err
	}

	// FIXME: connect with commit code and call refstore directly
	if newRef != nil {
		// TODO @jhowardmsft LCOW - Again, assume the OS of the host for now
		if err := daemon.TagImageWithReference(id, runtime.GOOS, newRef); err != nil {
			return err
		}
	}

	daemon.LogImageEvent(id.String(), id.String(), "import")
	outStream.Write(streamformatter.FormatStatus("", id.String()))
	return nil
}
