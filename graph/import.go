package graph

import (
	"io"
	"net/http"
	"net/url"

	"github.com/tiborvass/docker/pkg/archive"
	"github.com/tiborvass/docker/pkg/httputils"
	"github.com/tiborvass/docker/pkg/progressreader"
	"github.com/tiborvass/docker/pkg/streamformatter"
	"github.com/tiborvass/docker/runconfig"
	"github.com/tiborvass/docker/utils"
)

// Import imports an image, getting the archived layer data either from
// inConfig (if src is "-"), or from a URI specified in src. Progress output is
// written to outStream. Repository and tag names can optionally be given in
// the repo and tag arguments, respectively.
func (s *TagStore) Import(src string, repo string, tag string, inConfig io.ReadCloser, outStream io.Writer, containerConfig *runconfig.Config) error {
	var (
		sf      = streamformatter.NewJSONStreamFormatter()
		archive archive.Reader
		resp    *http.Response
	)

	if src == "-" {
		archive = inConfig
	} else {
		u, err := url.Parse(src)
		if err != nil {
			return err
		}
		if u.Scheme == "" {
			u.Scheme = "http"
			u.Host = src
			u.Path = ""
		}
		outStream.Write(sf.FormatStatus("", "Downloading from %s", u))
		resp, err = httputils.Download(u.String())
		if err != nil {
			return err
		}
		progressReader := progressreader.New(progressreader.Config{
			In:        resp.Body,
			Out:       outStream,
			Formatter: sf,
			Size:      resp.ContentLength,
			NewLines:  true,
			ID:        "",
			Action:    "Importing",
		})
		defer progressReader.Close()
		archive = progressReader
	}

	img, err := s.graph.Create(archive, "", "", "Imported from "+src, "", nil, containerConfig)
	if err != nil {
		return err
	}
	// Optionally register the image at REPO/TAG
	if repo != "" {
		if err := s.Tag(repo, tag, img.ID, true); err != nil {
			return err
		}
	}
	outStream.Write(sf.FormatStatus("", img.ID))
	logID := img.ID
	if tag != "" {
		logID = utils.ImageReference(logID, tag)
	}

	s.eventsService.Log("import", logID, "")
	return nil
}
