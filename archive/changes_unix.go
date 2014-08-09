// +build !windows

package archive

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/system"
)

type FileInfo struct {
	parent     *FileInfo
	name       string
	stat       syscall.Stat_t
	children   map[string]*FileInfo
	capability []byte
}

func (root *FileInfo) LookUp(path string) *FileInfo {
	parent := root
	if path == "/" {
		return root
	}

	pathElements := strings.Split(path, "/")
	for _, elem := range pathElements {
		if elem != "" {
			child := parent.children[elem]
			if child == nil {
				return nil
			}
			parent = child
		}
	}
	return parent
}

func (info *FileInfo) path() string {
	if info.parent == nil {
		return "/"
	}
	return filepath.Join(info.parent.path(), info.name)
}

func (info *FileInfo) isDir() bool {
	return info.parent == nil || info.stat.Mode&syscall.S_IFDIR == syscall.S_IFDIR
}

func (info *FileInfo) addChanges(oldInfo *FileInfo, changes *[]Change) {
	if oldInfo == nil {
		// add
		change := Change{
			Path: info.path(),
			Kind: ChangeAdd,
		}
		*changes = append(*changes, change)
	}

	// We make a copy so we can modify it to detect additions
	// also, we only recurse on the old dir if the new info is a directory
	// otherwise any previous delete/change is considered recursive
	oldChildren := make(map[string]*FileInfo)
	if oldInfo != nil && info.isDir() {
		for k, v := range oldInfo.children {
			oldChildren[k] = v
		}
	}

	for name, newChild := range info.children {
		oldChild, _ := oldChildren[name]
		if oldChild != nil {
			// change?
			oldStat := &oldChild.stat
			newStat := &newChild.stat
			// Note: We can't compare inode or ctime or blocksize here, because these change
			// when copying a file into a container. However, that is not generally a problem
			// because any content change will change mtime, and any status change should
			// be visible when actually comparing the stat fields. The only time this
			// breaks down is if some code intentionally hides a change by setting
			// back mtime
			if oldStat.Mode != newStat.Mode ||
				oldStat.Uid != newStat.Uid ||
				oldStat.Gid != newStat.Gid ||
				oldStat.Rdev != newStat.Rdev ||
				// Don't look at size for dirs, its not a good measure of change
				(oldStat.Size != newStat.Size && oldStat.Mode&syscall.S_IFDIR != syscall.S_IFDIR) ||
				!sameFsTime(system.GetLastModification(oldStat), system.GetLastModification(newStat)) ||
				bytes.Compare(oldChild.capability, newChild.capability) != 0 {
				change := Change{
					Path: newChild.path(),
					Kind: ChangeModify,
				}
				*changes = append(*changes, change)
			}

			// Remove from copy so we can detect deletions
			delete(oldChildren, name)
		}

		newChild.addChanges(oldChild, changes)
	}
	for _, oldChild := range oldChildren {
		// delete
		change := Change{
			Path: oldChild.path(),
			Kind: ChangeDelete,
		}
		*changes = append(*changes, change)
	}

}

func (info *FileInfo) Changes(oldInfo *FileInfo) []Change {
	var changes []Change

	info.addChanges(oldInfo, &changes)

	return changes
}

func newRootFileInfo() *FileInfo {
	root := &FileInfo{
		name:     "/",
		children: make(map[string]*FileInfo),
	}
	return root
}

func collectFileInfo(sourceDir string) (*FileInfo, error) {
	root := newRootFileInfo()

	err := filepath.Walk(sourceDir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Rebase path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.Join("/", relPath)

		if relPath == "/" {
			return nil
		}

		parent := root.LookUp(filepath.Dir(relPath))
		if parent == nil {
			return fmt.Errorf("collectFileInfo: Unexpectedly no parent for %s", relPath)
		}

		info := &FileInfo{
			name:     filepath.Base(relPath),
			children: make(map[string]*FileInfo),
			parent:   parent,
		}

		if err := syscall.Lstat(path, &info.stat); err != nil {
			return err
		}

		info.capability, _ = system.Lgetxattr(path, "security.capability")

		parent.children[info.name] = info

		return nil
	})
	if err != nil {
		return nil, err
	}
	return root, nil
}
