// +build linux

package fsutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"unsafe"
)

func locateDummyIfEmpty(path string) (string, error) {
	children, err := ioutil.ReadDir(path)
	if err != nil {
		return "", err
	}
	if len(children) != 0 {
		return "", nil
	}
	dummyFile, err := ioutil.TempFile(path, "fsutils-dummy")
	if err != nil {
		return "", err
	}
	name := dummyFile.Name()
	if err = dummyFile.Close(); err != nil {
		return name, err
	}
	return name, nil
}

// SupportsDType returns whether the filesystem mounted on path supports d_type
func SupportsDType(path string) (bool, error) {
	// locate dummy so that we have at least one dirent
	dummy, err := locateDummyIfEmpty(path)
	if err != nil {
		return false, err
	}
	if dummy != "" {
		defer os.Remove(dummy)
	}

	visited := 0
	supportsDType := true
	fn := func(ent *syscall.Dirent) bool {
		visited++
		if ent.Type == syscall.DT_UNKNOWN {
			supportsDType = false
			// stop iteration
			return true
		}
		// continue iteration
		return false
	}
	if err = iterateReadDir(path, fn); err != nil {
		return false, err
	}
	if visited == 0 {
		return false, fmt.Errorf("did not hit any dirent during iteration %s", path)
	}
	return supportsDType, nil
}

func iterateReadDir(path string, fn func(*syscall.Dirent) bool) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	fd := int(d.Fd())
	buf := make([]byte, 4096)
	for {
		nbytes, err := syscall.ReadDirent(fd, buf)
		if err != nil {
			return err
		}
		if nbytes == 0 {
			break
		}
		for off := 0; off < nbytes; {
			ent := (*syscall.Dirent)(unsafe.Pointer(&buf[off]))
			if stop := fn(ent); stop {
				return nil
			}
			off += int(ent.Reclen)
		}
	}
	return nil
}
