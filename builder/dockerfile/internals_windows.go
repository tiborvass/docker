package dockerfile

import "github.com/tiborvass/docker/pkg/idtools"

func parseChownFlag(chown, ctrRootPath string, idMappings *idtools.IDMappings) (idtools.IDPair, error) {
	return idMappings.RootPair(), nil
}
