package builder

import "path/filepath"

type FsContext string

func (c FsContext) Path() string {
	return string(c)
}

func (c FsContext) Walk(walkFn filepath.WalkFunc) error {
	return filepath.Walk(string(c), walkFn)
}
