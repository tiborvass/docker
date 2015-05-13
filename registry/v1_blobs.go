package registry

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/pkg/jsonmessage"
)

type blobs struct {
	*repository
}

// server

func (b *blobs) ServeBlob(ctx context.Context, w http.ResponseWriter, r *http.Request, dgst digest.Digest) error {
	panic("not implemented")
}

// statter

func (b *blobs) Stat(ctx context.Context, dgst digest.Digest) (desc distribution.Descriptor, err error) {
	imgID := digestToImgID(dgst)

	// hack: we're getting the size from a previous call
	// due to v1's architecture
	size, ok := b.sizes[imgID]
	if !ok {
		err = fmt.Errorf("could not find size information for %s", imgID)
		return
	}

	var (
		success  bool
		notExist bool
	)

	if success, err = b.loopEndpoints(false, func(endpoint string) error {
		err := b.session.LookupRemoteImage(imgID, endpoint)
		if err == nil {
			b.currentEndpoint = endpoint
			return nil
		}
		if _, ok := err.(*jsonmessage.JSONError); ok {
			notExist = true
			b.currentEndpoint = endpoint
			return nil
		}
		// unexpected, try another endpoint
		return err
	}); !success {
		return
	}

	if notExist {
		err = distribution.ErrBlobUnknown
		return
	}

	return distribution.Descriptor{
		// TODO(tiborvass): what to do with different compression formats?
		MediaType: "application/x-tar",
		Length:    size,
		Digest:    dgst,
	}, nil
}

// provider

func (b *blobs) Get(ctx context.Context, dgst digest.Digest) ([]byte, error) {
	r, err := b.Open(ctx, dgst)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func (b *blobs) Open(ctx context.Context, dgst digest.Digest) (distribution.ReadSeekCloser, error) {
	var (
		blob  distribution.ReadSeekCloser
		imgID = digestToImgID(dgst)
	)

	loopFn := func(endpoint string) error {
		// hack: we're getting the size from a previous call
		// due to v1's architecture
		size, ok := b.sizes[imgID]
		if !ok {
			return fmt.Errorf("could not find size information for %s", imgID)
		}

		if size > 0 {
			u := fmt.Sprintf("%simages/%s/layer", endpoint, imgID)
			blob = transport.NewHTTPReadSeeker(b.session.client, u, size)
			b.currentEndpoint = endpoint
			return nil
		}

		resp, err := b.session.GetRemoteImageLayer(imgID, endpoint, size)
		blob = noSeeker{resp}
		if err == nil {
			b.currentEndpoint = endpoint
		}
		return err
	}

	if success, errs := b.loopEndpoints(true, loopFn); !success {
		return nil, errs
	}

	return blob, nil
}

// noSeeker implements io.Seeker but returns an error on Seek()
type noSeeker struct{ io.ReadCloser }

func (n noSeeker) Seek(offset int64, whence int) (int64, error) {
	return -1, errors.New("seek not supported")
}

// ingester

func (b *blobs) Put(ctx context.Context, mediaType string, p []byte) (desc distribution.Descriptor, err error) {
	var bw distribution.BlobWriter
	bw, err = b.Create(ctx)
	if err != nil {
		return
	}
	_, err = bw.Write(p)
	if err != nil {
		return
	}
	// TODO(tiborvass): if the digest is an imageID, are we losing that information somewhere?
	return bw.Commit(ctx, distribution.Descriptor{MediaType: mediaType})
}

func (b *blobs) Create(ctx context.Context) (distribution.BlobWriter, error) {
	if b.currentEndpoint != "" {
		return nil, errors.New("Internal endpoint not set, make sure you call Stat or Open before Create")
	}
	return &blobWriter{
		blobs: b,
		// TODO: fixme
		id:        "randomid",
		buf:       new(bytes.Buffer),
		startTime: time.Now(),
	}, nil
}
func (b *blobs) Resume(ctx context.Context, id string) (distribution.BlobWriter, error) {
	return nil, errors.New("not implemented")
}

type blobWriter struct {
	*blobs
	reader    io.Reader
	buf       *bytes.Buffer
	id        string
	startTime time.Time
}

func (bw *blobWriter) Read(p []byte) (n int, err error) {
	return bw.buf.Read(p)
}

func (bw *blobWriter) Write(p []byte) (n int, err error) {
	return bw.buf.Write(p)
}

func (bw *blobWriter) ReadFrom(r io.Reader) (n int64, err error) {
	bw.reader = r
	return 0, nil
}

func (bw *blobWriter) Close() error {
	bw.buf = nil
	return nil
}

func (bw *blobWriter) ID() string {
	return bw.id
}

func (bw *blobWriter) StartedAt() time.Time {
	return bw.startTime
}

func (bw *blobWriter) Commit(ctx context.Context, provisional distribution.Descriptor) (canonical distribution.Descriptor, err error) {
	imgID := digestToImgID(provisional.Digest)
	jsonData, ok := bw.jsonData[imgID]
	if !ok {
		err = fmt.Errorf("Could not find json metadata for image %s. Make sure you call PreparePush", imgID)
		return
	}
	imgData := &ImgData{ID: imgID}
	// Send the json
	err = bw.session.PushImageJSONRegistry(imgData, jsonData, bw.currentEndpoint)
	if err != nil {
		return
	}

	var reader io.Reader
	if bw.reader != nil {
		// from ReadFrom
		reader = bw.reader
	} else {
		// from Write
		reader = bw.buf
	}

	imgData.Checksum, imgData.ChecksumPayload, err = bw.session.PushImageLayerRegistry(imgID, reader, bw.currentEndpoint, jsonData)
	if err != nil {
		return
	}

	// Send the checksum
	err = bw.session.PushImageChecksumRegistry(imgData, bw.currentEndpoint)
	if err != nil {
		return
	}

	return provisional, nil
}

func (bw *blobWriter) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("seek not implemented")
}

func (bw *blobWriter) Cancel(ctx context.Context) error {
	return errors.New("not implemented")
}
