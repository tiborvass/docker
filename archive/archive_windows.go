package archive

import (
	"bufio"
	"io"

	"github.com/docker/docker/vendor/src/code.google.com/p/go/src/pkg/archive/tar"
)

func addTarFile(path, name string, tw *tar.Writer, twBuf *bufio.Writer) error {
	return ErrNotImplemented
}

func createTarFile(path, extractDir string, hdr *tar.Header, reader io.Reader, Lchown bool) error {
	return ErrNotImplemented
}

func Untar(archive io.Reader, dest string, options *TarOptions) error {
	return ErrNotImplemented
}
