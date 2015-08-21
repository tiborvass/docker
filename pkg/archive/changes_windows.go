package archive

import (
	"github.com/tiborvass/docker/pkg/system"
)

func statDifferent(oldStat *system.StatT, newStat *system.StatT) bool {

	// Don't look at size for dirs, its not a good measure of change
	if oldStat.ModTime() != newStat.ModTime() ||
		oldStat.Mode() != newStat.Mode() ||
		oldStat.Size() != newStat.Size() && !oldStat.IsDir() {
		return true
	}
	return false
}

func (info *FileInfo) isDir() bool {
	return info.parent == nil || info.stat.IsDir()
}
