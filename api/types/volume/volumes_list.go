package volume

// ----------------------------------------------------------------------------
// DO NOT EDIT THIS FILE
// This file was generated by `swagger generate operation`
//
// See hack/generate-swagger-api.sh
// ----------------------------------------------------------------------------

import "github.com/moby/moby-core/api/types"

// VolumesListOKBody volumes list o k body
// swagger:model VolumesListOKBody
type VolumesListOKBody struct {

	// List of volumes
	// Required: true
	Volumes []*types.Volume `json:"Volumes"`

	// Warnings that occurred when fetching the list of volumes
	// Required: true
	Warnings []string `json:"Warnings"`
}
