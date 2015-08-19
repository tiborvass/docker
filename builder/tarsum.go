package builder

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/chrootarchive"
	"github.com/docker/docker/pkg/tarsum"
)

type tarSumContext struct {
	root string
	sums tarsum.FileInfoSums
}

func (c *tarSumContext) Get(path string) (ContextEntry, error) {
	fmt.Println("toto get", path)
	fullpath := filepath.Join(c.root, path)
	fmt.Println("toto fullpath", fullpath)
	fi, err := os.Lstat(fullpath)
	if err != nil {
		return nil, err
	}
	tsInfo := c.sums.GetFile(path)
	hfi := HashFileInfo{FileInfo: FileInfo{fi, fullpath}, Hash: tsInfo.Sum()}
	return hfi, nil
}

func MakeTarSumContext(tarStream io.ReadCloser) (Context, error) {
	defer tarStream.Close()

	root, err := ioutil.TempDir("", "docker-builder")
	if err != nil {
		return nil, err
	}

	// Make sure we clean-up upon error.  In the happy case the caller
	// is expected to manage the clean-up
	defer func() {
		if err != nil {
			if e := os.RemoveAll(root); e != nil {
				logrus.Debugf("[BUILDER] failed to remove temporary context: %s", e)
			}
		}
	}()

	decompressedStream, err := archive.DecompressStream(tarStream)
	if err != nil {
		return nil, err
	}

	sum, err := tarsum.NewTarSum(decompressedStream, true, tarsum.Version1)
	if err != nil {
		return nil, err
	}

	if err := chrootarchive.Untar(sum, root, nil); err != nil {
		return nil, err
	}

	return HelperContext{&tarSumContext{root, sum.GetSums()}}, nil
}

func (c *tarSumContext) Walk(walkFn filepath.WalkFunc) error {
	for _, tsInfo := range c.sums {
		path := filepath.Join(c.root, tsInfo.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		hfi := HashFileInfo{FileInfo: FileInfo{info, path}, Hash: tsInfo.Sum()}
		if err := walkFn(path, hfi, nil); err != nil {
			return err
		}
	}
	return nil
}
