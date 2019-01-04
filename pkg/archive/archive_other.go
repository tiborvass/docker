// +build !linux

package archive // import "github.com/tiborvass/docker/pkg/archive"

func getWhiteoutConverter(format WhiteoutFormat, inUserNS bool) tarWhiteoutConverter {
	return nil
}
