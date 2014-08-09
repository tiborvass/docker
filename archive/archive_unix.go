// +build !windows

package archive

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/system"
	"github.com/docker/docker/utils"
	"github.com/docker/docker/vendor/src/code.google.com/p/go/src/pkg/archive/tar"
)

func addTarFile(path, name string, tw *tar.Writer, twBuf *bufio.Writer) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	link := ""
	if fi.Mode()&os.ModeSymlink != 0 {
		if link, err = os.Readlink(path); err != nil {
			return err
		}
	}

	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return err
	}

	if fi.IsDir() && !strings.HasSuffix(name, "/") {
		name = name + "/"
	}

	hdr.Name = name

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if ok {
		// Currently go does not fill in the major/minors
		if stat.Mode&syscall.S_IFBLK == syscall.S_IFBLK ||
			stat.Mode&syscall.S_IFCHR == syscall.S_IFCHR {
			hdr.Devmajor = int64(major(uint64(stat.Rdev)))
			hdr.Devminor = int64(minor(uint64(stat.Rdev)))
		}
	}

	capability, _ := system.Lgetxattr(path, "security.capability")
	if capability != nil {
		hdr.Xattrs = make(map[string]string)
		hdr.Xattrs["security.capability"] = string(capability)
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	if hdr.Typeflag == tar.TypeReg {
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		twBuf.Reset(tw)
		_, err = io.Copy(twBuf, file)
		file.Close()
		if err != nil {
			return err
		}
		err = twBuf.Flush()
		if err != nil {
			return err
		}
		twBuf.Reset(nil)
	}

	return nil
}

func createTarFile(path, extractDir string, hdr *tar.Header, reader io.Reader, Lchown bool) error {
	// hdr.Mode is in linux format, which we can use for sycalls,
	// but for os.Foo() calls we need the mode converted to os.FileMode,
	// so use hdrInfo.Mode() (they differ for e.g. setuid bits)
	hdrInfo := hdr.FileInfo()

	switch hdr.Typeflag {
	case tar.TypeDir:
		// Create directory unless it exists as a directory already.
		// In that case we just want to merge the two
		if fi, err := os.Lstat(path); !(err == nil && fi.IsDir()) {
			if err := os.Mkdir(path, hdrInfo.Mode()); err != nil {
				return err
			}
		}

	case tar.TypeReg, tar.TypeRegA:
		// Source is regular file
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, hdrInfo.Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(file, reader); err != nil {
			file.Close()
			return err
		}
		file.Close()

	case tar.TypeBlock, tar.TypeChar, tar.TypeFifo:
		mode := uint32(hdr.Mode & 07777)
		switch hdr.Typeflag {
		case tar.TypeBlock:
			mode |= syscall.S_IFBLK
		case tar.TypeChar:
			mode |= syscall.S_IFCHR
		case tar.TypeFifo:
			mode |= syscall.S_IFIFO
		}

		if err := syscall.Mknod(path, mode, int(mkdev(hdr.Devmajor, hdr.Devminor))); err != nil {
			return err
		}

	case tar.TypeLink:
		if err := os.Link(filepath.Join(extractDir, hdr.Linkname), path); err != nil {
			return err
		}

	case tar.TypeSymlink:
		if err := os.Symlink(hdr.Linkname, path); err != nil {
			return err
		}

	case tar.TypeXGlobalHeader:
		utils.Debugf("PAX Global Extended Headers found and ignored")
		return nil

	default:
		return fmt.Errorf("Unhandled tar header type %d\n", hdr.Typeflag)
	}

	if err := os.Lchown(path, hdr.Uid, hdr.Gid); err != nil && Lchown {
		return err
	}

	for key, value := range hdr.Xattrs {
		if err := system.Lsetxattr(path, key, []byte(value), 0); err != nil {
			return err
		}
	}

	// There is no LChmod, so ignore mode for symlink. Also, this
	// must happen after chown, as that can modify the file mode
	if hdr.Typeflag != tar.TypeSymlink {
		if err := os.Chmod(path, hdrInfo.Mode()); err != nil {
			return err
		}
	}

	ts := []syscall.Timespec{timeToTimespec(hdr.AccessTime), timeToTimespec(hdr.ModTime)}
	// syscall.UtimesNano doesn't support a NOFOLLOW flag atm, and
	if hdr.Typeflag != tar.TypeSymlink {
		if err := system.UtimesNano(path, ts); err != nil && err != system.ErrNotSupportedPlatform {
			return err
		}
	} else {
		if err := system.LUtimesNano(path, ts); err != nil && err != system.ErrNotSupportedPlatform {
			return err
		}
	}
	return nil
}

// Untar reads a stream of bytes from `archive`, parses it as a tar archive,
// and unpacks it into the directory at `path`.
// The archive may be compressed with one of the following algorithms:
//  identity (uncompressed), gzip, bzip2, xz.
// FIXME: specify behavior when target path exists vs. doesn't exist.
func Untar(archive io.Reader, dest string, options *TarOptions) error {
	if archive == nil {
		return fmt.Errorf("Empty archive")
	}

	decompressedArchive, err := DecompressStream(archive)
	if err != nil {
		return err
	}
	defer decompressedArchive.Close()

	tr := tar.NewReader(decompressedArchive)
	trBuf := bufio.NewReaderSize(nil, trBufSize)

	var dirs []*tar.Header

	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return err
		}

		// Normalize name, for safety and for a simple is-root check
		hdr.Name = filepath.Clean(hdr.Name)

		if !strings.HasSuffix(hdr.Name, "/") {
			// Not the root directory, ensure that the parent directory exists
			parent := filepath.Dir(hdr.Name)
			parentPath := filepath.Join(dest, parent)
			if _, err := os.Lstat(parentPath); err != nil && os.IsNotExist(err) {
				err = os.MkdirAll(parentPath, 0777)
				if err != nil {
					return err
				}
			}
		}

		path := filepath.Join(dest, hdr.Name)

		// If path exits we almost always just want to remove and replace it
		// The only exception is when it is a directory *and* the file from
		// the layer is also a directory. Then we want to merge them (i.e.
		// just apply the metadata from the layer).
		if fi, err := os.Lstat(path); err == nil {
			if fi.IsDir() && hdr.Name == "." {
				continue
			}
			if !(fi.IsDir() && hdr.Typeflag == tar.TypeDir) {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
			}
		}
		trBuf.Reset(tr)
		if err := createTarFile(path, dest, hdr, trBuf, options == nil || !options.NoLchown); err != nil {
			return err
		}

		// Directory mtimes must be handled at the end to avoid further
		// file creation in them to modify the directory mtime
		if hdr.Typeflag == tar.TypeDir {
			dirs = append(dirs, hdr)
		}
	}

	for _, hdr := range dirs {
		path := filepath.Join(dest, hdr.Name)
		ts := []syscall.Timespec{timeToTimespec(hdr.AccessTime), timeToTimespec(hdr.ModTime)}
		if err := syscall.UtimesNano(path, ts); err != nil {
			return err
		}
	}

	return nil
}
