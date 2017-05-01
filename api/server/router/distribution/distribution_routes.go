package distribution

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/tiborvass/docker/api/server/httputils"
	"github.com/tiborvass/docker/api/types"
	registrytypes "github.com/tiborvass/docker/api/types/registry"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func (s *distributionRouter) getDistributionInfo(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := httputils.ParseForm(r); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	var (
		config              = &types.AuthConfig{}
		authEncoded         = r.Header.Get("X-Registry-Auth")
		distributionInspect registrytypes.DistributionInspect
	)

	if authEncoded != "" {
		authJSON := base64.NewDecoder(base64.URLEncoding, strings.NewReader(authEncoded))
		if err := json.NewDecoder(authJSON).Decode(&config); err != nil {
			// for a search it is not an error if no auth was given
			// to increase compatibility with the existing api it is defaulting to be empty
			config = &types.AuthConfig{}
		}
	}

	image := vars["name"]

	ref, err := reference.ParseAnyReference(image)
	if err != nil {
		return err
	}
	namedRef, ok := ref.(reference.Named)
	if !ok {
		if _, ok := ref.(reference.Digested); ok {
			// full image ID
			return errors.Errorf("no manifest found for full image ID")
		}
		return errors.Errorf("unknown image reference format: %s", image)
	}

	distrepo, _, err := s.backend.GetRepository(ctx, namedRef, config)
	if err != nil {
		return err
	}

	if canonicalRef, ok := namedRef.(reference.Canonical); !ok {
		namedRef = reference.TagNameOnly(namedRef)

		taggedRef, ok := namedRef.(reference.NamedTagged)
		if !ok {
			return errors.Errorf("image reference not tagged: %s", image)
		}

		dscrptr, err := distrepo.Tags(ctx).Get(ctx, taggedRef.Tag())
		if err != nil {
			return err
		}
		distributionInspect.Digest = dscrptr.Digest
	} else {
		distributionInspect.Digest = canonicalRef.Digest()
	}
	// at this point, we have a digest, so we can retrieve the manifest

	mnfstsrvc, err := distrepo.Manifests(ctx)
	if err != nil {
		return err
	}
	mnfst, err := mnfstsrvc.Get(ctx, distributionInspect.Digest)
	if err != nil {
		return err
	}

	// retrieve platform information depending on the type of manifest
	switch mnfstObj := mnfst.(type) {
	case *manifestlist.DeserializedManifestList:
		for _, m := range mnfstObj.Manifests {
			distributionInspect.Platforms = append(distributionInspect.Platforms, m.Platform)
		}
	case *schema2.DeserializedManifest:
		blobsrvc := distrepo.Blobs(ctx)
		configJSON, err := blobsrvc.Get(ctx, mnfstObj.Config.Digest)
		var platform manifestlist.PlatformSpec
		if err == nil {
			err := json.Unmarshal(configJSON, &platform)
			if err == nil {
				distributionInspect.Platforms = append(distributionInspect.Platforms, platform)
			}
		}
	case *schema1.SignedManifest:
		platform := manifestlist.PlatformSpec{
			Architecture: mnfstObj.Architecture,
			OS:           "linux",
		}
		distributionInspect.Platforms = append(distributionInspect.Platforms, platform)
	}

	return httputils.WriteJSON(w, http.StatusOK, distributionInspect)
}
