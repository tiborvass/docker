package archive

type FileInfo struct {
	parent     *FileInfo
	name       string
	stat       interface{}
	children   map[string]*FileInfo
	capability []byte
}

func (info *FileInfo) Changes(oldInfo *FileInfo) []Change {
	return nil
}

func (root *FileInfo) LookUp(path string) *FileInfo {
	return nil
}

func collectFileInfo(sourceDir string) (*FileInfo, error) {
	return nil, ErrNotImplemented
}
