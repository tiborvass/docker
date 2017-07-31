package dockerfile

import "github.com/moby/moby-core/pkg/idtools"

func fixPermissions(source, destination string, rootIDs idtools.IDPair) error {
	// chown is not supported on Windows
	return nil
}
