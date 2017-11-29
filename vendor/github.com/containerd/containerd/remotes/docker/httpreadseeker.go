package docker

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/pkg/errors"
)

type httpReadSeeker struct {
	size   int64
	offset int64
	rc     io.ReadCloser
	open   func(offset int64) (io.ReadCloser, error)
	closed bool
}

func newHTTPReadSeeker(size int64, open func(offset int64) (io.ReadCloser, error)) (io.ReadCloser, error) {
	return &httpReadSeeker{
		size: size,
		open: open,
	}, nil
}

func (hrs *httpReadSeeker) Read(p []byte) (n int, err error) {
	if hrs.closed {
		return 0, io.EOF
	}

	rd, err := hrs.reader()
	if err != nil {
		return 0, err
	}

	n, err = rd.Read(p)
	hrs.offset += int64(n)
	return
}

func (hrs *httpReadSeeker) Close() error {
	if hrs.closed {
		return nil
	}
	hrs.closed = true
	if hrs.rc != nil {
		return hrs.rc.Close()
	}

	return nil
}

func (hrs *httpReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if hrs.closed {
		return 0, errors.Wrap(errdefs.ErrUnavailable, "Fetcher.Seek: closed")
	}

	abs := hrs.offset
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs += offset
	case io.SeekEnd:
		if hrs.size == -1 {
			return 0, errors.Wrap(errdefs.ErrUnavailable, "Fetcher.Seek: unknown size, cannot seek from end")
		}
		abs = hrs.size + offset
	default:
		return 0, errors.Wrap(errdefs.ErrInvalidArgument, "Fetcher.Seek: invalid whence")
	}

	if abs < 0 {
		return 0, errors.Wrapf(errdefs.ErrInvalidArgument, "Fetcher.Seek: negative offset")
	}

	if abs != hrs.offset {
		if hrs.rc != nil {
			if err := hrs.rc.Close(); err != nil {
				log.L.WithError(err).Errorf("Fetcher.Seek: failed to close ReadCloser")
			}

			hrs.rc = nil
		}

		hrs.offset = abs
	}

	return hrs.offset, nil
}

func (hrs *httpReadSeeker) reader() (io.Reader, error) {
	if hrs.rc != nil {
		return hrs.rc, nil
	}

	if hrs.size == -1 || hrs.offset < hrs.size {
		// only try to reopen the body request if we are seeking to a value
		// less than the actual size.
		if hrs.open == nil {
			return nil, errors.Wrapf(errdefs.ErrNotImplemented, "cannot open")
		}

		rc, err := hrs.open(hrs.offset)
		if err != nil {
			return nil, errors.Wrapf(err, "httpReaderSeeker: failed open")
		}

		if hrs.rc != nil {
			if err := hrs.rc.Close(); err != nil {
				log.L.WithError(err).Errorf("httpReadSeeker: failed to close ReadCloser")
			}
		}
		hrs.rc = rc
	} else {
		// There is an edge case here where offset == size of the content. If
		// we seek, we will probably get an error for content that cannot be
		// sought (?). In that case, we should err on committing the content,
		// as the length is already satisified but we just return the empty
		// reader instead.

		hrs.rc = ioutil.NopCloser(bytes.NewReader([]byte{}))
	}

	return hrs.rc, nil
}
