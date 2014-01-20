package archive

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"github.com/dotcloud/docker/utils"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

type Archive io.Reader

type Compression int

type TarOptions struct {
	Includes    []string
	Excludes    []string
	Recursive   bool
	Compression Compression
	CreateFiles []string
}

const (
	Uncompressed Compression = iota
	Bzip2
	Gzip
	Xz
)

func DetectCompression(source []byte) Compression {
	sourceLen := len(source)
	for compression, m := range map[Compression][]byte{
		Bzip2: {0x42, 0x5A, 0x68},
		Gzip:  {0x1F, 0x8B, 0x08},
		Xz:    {0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
	} {
		fail := false
		if len(m) > sourceLen {
			utils.Debugf("Len too short")
			continue
		}
		i := 0
		for _, b := range m {
			if b != source[i] {
				fail = true
				break
			}
			i++
		}
		if !fail {
			return compression
		}
	}
	return Uncompressed
}

func xzDecompress(archive io.Reader) (io.Reader, error) {
	args := []string{"xz", "-d", "-c", "-q"}

	return CmdStream(exec.Command(args[0], args[1:]...), archive, nil)
}

func DecompressStream(archive io.Reader) (io.Reader, error) {
	buf := make([]byte, 10)
	totalN := 0
	for totalN < 10 {
		n, err := archive.Read(buf[totalN:])
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("Tarball too short")
			}
			return nil, err
		}
		totalN += n
		utils.Debugf("[tar autodetect] n: %d", n)
	}
	compression := DetectCompression(buf)
	wrap := io.MultiReader(bytes.NewReader(buf), archive)

	switch compression {
	case Uncompressed:
		return wrap, nil
	case Gzip:
		return gzip.NewReader(wrap)
	case Bzip2:
		return bzip2.NewReader(wrap), nil
	case Xz:
		return xzDecompress(wrap)
	default:
		return nil, fmt.Errorf("Unsupported compression format %s", (&compression).Extension())
	}
}

func (compression *Compression) Flag() string {
	switch *compression {
	case Bzip2:
		return "j"
	case Gzip:
		return "z"
	case Xz:
		return "J"
	}
	return ""
}

func (compression *Compression) Extension() string {
	switch *compression {
	case Uncompressed:
		return "tar"
	case Bzip2:
		return "tar.bz2"
	case Gzip:
		return "tar.gz"
	case Xz:
		return "tar.xz"
	}
	return ""
}

func createTarFile(path, extractDir string, hdr *tar.Header, reader *tar.Reader) error {
	switch hdr.Typeflag {
	case tar.TypeDir:
		// Create directory unless it exists as a directory already.
		// In that case we just want to merge the two
		if fi, err := os.Lstat(path); !(err == nil && fi.IsDir()) {
			if err := os.Mkdir(path, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		}

	case tar.TypeReg, tar.TypeRegA:
		// Source is regular file
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
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

	default:
		return fmt.Errorf("Unhandled tar header type %d\n", hdr.Typeflag)
	}

	if err := syscall.Lchown(path, hdr.Uid, hdr.Gid); err != nil {
		return err
	}

	// There is no LChmod, so ignore mode for symlink. Also, this
	// must happen after chown, as that can modify the file mode
	if hdr.Typeflag != tar.TypeSymlink {
		if err := syscall.Chmod(path, uint32(hdr.Mode&07777)); err != nil {
			return err
		}
	}

	ts := []syscall.Timespec{timeToTimespec(hdr.AccessTime), timeToTimespec(hdr.ModTime)}
	// syscall.UtimesNano doesn't support a NOFOLLOW flag atm, and
	if hdr.Typeflag != tar.TypeSymlink {
		if err := syscall.UtimesNano(path, ts); err != nil {
			return err
		}
	} else {
		if err := LUtimesNano(path, ts); err != nil {
			return err
		}
	}
	return nil
}

// Tar creates an archive from the directory at `path`, and returns it as a
// stream of bytes.
func Tar(path string, compression Compression) (io.Reader, error) {
	return TarFilter(path, &TarOptions{Recursive: true, Compression: compression})
}

func escapeName(name string) string {
	escaped := make([]byte, 0)
	for i, c := range []byte(name) {
		if i == 0 && c == '/' {
			continue
		}
		// all printable chars except "-" which is 0x2d
		if (0x20 <= c && c <= 0x7E) && c != 0x2d {
			escaped = append(escaped, c)
		} else {
			escaped = append(escaped, fmt.Sprintf("\\%03o", c)...)
		}
	}
	return string(escaped)
}

// Tar creates an archive from the directory at `path`, only including files whose relative
// paths are included in `filter`. If `filter` is nil, then all files are included.
func TarFilter(path string, options *TarOptions) (io.Reader, error) {
	args := []string{"tar", "--numeric-owner", "-f", "-", "-C", path, "-T", "-"}
	if options.Includes == nil {
		options.Includes = []string{"."}
	}
	args = append(args, "-c"+options.Compression.Flag())

	for _, exclude := range options.Excludes {
		args = append(args, fmt.Sprintf("--exclude=%s", exclude))
	}

	if !options.Recursive {
		args = append(args, "--no-recursion")
	}

	files := ""
	for _, f := range options.Includes {
		files = files + escapeName(f) + "\n"
	}

	tmpDir := ""

	if options.CreateFiles != nil {
		var err error // Can't use := here or we override the outer tmpDir
		tmpDir, err = ioutil.TempDir("", "docker-tar")
		if err != nil {
			return nil, err
		}

		files = files + "-C" + tmpDir + "\n"
		for _, f := range options.CreateFiles {
			path := filepath.Join(tmpDir, f)
			err := os.MkdirAll(filepath.Dir(path), 0600)
			if err != nil {
				return nil, err
			}

			if file, err := os.OpenFile(path, os.O_CREATE, 0600); err != nil {
				return nil, err
			} else {
				file.Close()
			}
			files = files + escapeName(f) + "\n"
		}
	}

	return CmdStream(exec.Command(args[0], args[1:]...), bytes.NewBufferString(files), func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})
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

	archive, err := DecompressStream(archive)
	if err != nil {
		return err
	}

	tr := tar.NewReader(archive)

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

		if options != nil {
			excludeFile := false
			for _, exclude := range options.Excludes {
				if strings.HasPrefix(hdr.Name, exclude) {
					excludeFile = true
					break
				}
			}
			if excludeFile {
				continue
			}
		}

		// Normalize name, for safety and for a simple is-root check
		hdr.Name = filepath.Clean(hdr.Name)

		if !strings.HasSuffix(hdr.Name, "/") {
			// Not the root directory, ensure that the parent directory exists
			parent := filepath.Dir(hdr.Name)
			parentPath := filepath.Join(dest, parent)
			if _, err := os.Lstat(parentPath); err != nil && os.IsNotExist(err) {
				err = os.MkdirAll(parentPath, 600)
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
			if !(fi.IsDir() && hdr.Typeflag == tar.TypeDir) {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
			}
		}

		if err := createTarFile(path, dest, hdr, tr); err != nil {
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

// TarUntar is a convenience function which calls Tar and Untar, with
// the output of one piped into the other. If either Tar or Untar fails,
// TarUntar aborts and returns the error.
func TarUntar(src string, filter []string, dst string) error {
	utils.Debugf("TarUntar(%s %s %s)", src, filter, dst)
	archive, err := TarFilter(src, &TarOptions{Compression: Uncompressed, Includes: filter, Recursive: true})
	if err != nil {
		return err
	}
	return Untar(archive, dst, nil)
}

// UntarPath is a convenience function which looks for an archive
// at filesystem path `src`, and unpacks it at `dst`.
func UntarPath(src, dst string) error {
	if archive, err := os.Open(src); err != nil {
		return err
	} else if err := Untar(archive, dst, nil); err != nil {
		return err
	}
	return nil
}

// CopyWithTar creates a tar archive of filesystem path `src`, and
// unpacks it at filesystem path `dst`.
// The archive is streamed directly with fixed buffering and no
// intermediary disk IO.
//
func CopyWithTar(src, dst string) error {
	srcSt, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcSt.IsDir() {
		return CopyFileWithTar(src, dst)
	}
	// Create dst, copy src's content into it
	utils.Debugf("Creating dest directory: %s", dst)
	if err := os.MkdirAll(dst, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	utils.Debugf("Calling TarUntar(%s, %s)", src, dst)
	return TarUntar(src, nil, dst)
}

// CopyFileWithTar emulates the behavior of the 'cp' command-line
// for a single file. It copies a regular file from path `src` to
// path `dst`, and preserves all its metadata.
//
// If `dst` ends with a trailing slash '/', the final destination path
// will be `dst/base(src)`.
func CopyFileWithTar(src, dst string) (err error) {
	utils.Debugf("CopyFileWithTar(%s, %s)", src, dst)
	srcSt, err := os.Stat(src)
	if err != nil {
		return err
	}
	if srcSt.IsDir() {
		return fmt.Errorf("Can't copy a directory")
	}
	// Clean up the trailing /
	if dst[len(dst)-1] == '/' {
		dst = path.Join(dst, filepath.Base(src))
	}
	// Create the holding directory if necessary
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil && !os.IsExist(err) {
		return err
	}

	r, w := io.Pipe()
	errC := utils.Go(func() error {
		defer w.Close()

		srcF, err := os.Open(src)
		if err != nil {
			return err
		}
		defer srcF.Close()

		tw := tar.NewWriter(w)
		hdr, err := tar.FileInfoHeader(srcSt, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.Base(dst)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := io.Copy(tw, srcF); err != nil {
			return err
		}
		tw.Close()
		return nil
	})
	defer func() {
		if er := <-errC; err != nil {
			err = er
		}
	}()
	return Untar(r, filepath.Dir(dst), nil)
}

// CmdStream executes a command, and returns its stdout as a stream.
// If the command fails to run or doesn't complete successfully, an error
// will be returned, including anything written on stderr.
func CmdStream(cmd *exec.Cmd, input io.Reader, atEnd func()) (io.Reader, error) {
	if input != nil {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			if atEnd != nil {
				atEnd()
			}
			return nil, err
		}
		// Write stdin if any
		go func() {
			io.Copy(stdin, input)
			stdin.Close()
		}()
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if atEnd != nil {
			atEnd()
		}
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		if atEnd != nil {
			atEnd()
		}
		return nil, err
	}
	pipeR, pipeW := io.Pipe()
	errChan := make(chan []byte)
	// Collect stderr, we will use it in case of an error
	go func() {
		errText, e := ioutil.ReadAll(stderr)
		if e != nil {
			errText = []byte("(...couldn't fetch stderr: " + e.Error() + ")")
		}
		errChan <- errText
	}()
	// Copy stdout to the returned pipe
	go func() {
		_, err := io.Copy(pipeW, stdout)
		if err != nil {
			pipeW.CloseWithError(err)
		}
		errText := <-errChan
		if err := cmd.Wait(); err != nil {
			pipeW.CloseWithError(fmt.Errorf("%s: %s", err, errText))
		} else {
			pipeW.Close()
		}
		if atEnd != nil {
			atEnd()
		}
	}()
	// Run the command and return the pipe
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return pipeR, nil
}

// NewTempArchive reads the content of src into a temporary file, and returns the contents
// of that file as an archive. The archive can only be read once - as soon as reading completes,
// the file will be deleted.
func NewTempArchive(src Archive, dir string) (*TempArchive, error) {
	f, err := ioutil.TempFile(dir, "")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, src); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()
	return &TempArchive{f, size}, nil
}

type TempArchive struct {
	*os.File
	Size int64 // Pre-computed from Stat().Size() as a convenience
}

func (archive *TempArchive) Read(data []byte) (int, error) {
	n, err := archive.File.Read(data)
	if err != nil {
		os.Remove(archive.File.Name())
	}
	return n, err
}
