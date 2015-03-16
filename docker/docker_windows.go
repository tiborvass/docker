// +build windows

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

func main() {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	dllPath := filepath.Join(tmpdir, ansiDll)
	RestoreAsset(tmpdir, ansiDll)
	dll, err := syscall.LoadDLL(dllPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	defer func() {
		dll.Release()
		os.RemoveAll(tmpdir)
	}()
	Main()
}
