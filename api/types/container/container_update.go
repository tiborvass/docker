package container // import "github.com/tiborvass/docker/api/types/container"

// ----------------------------------------------------------------------------
// Code generated by `swagger generate operation`. DO NOT EDIT.
//
// See hack/generate-swagger-api.sh
// ----------------------------------------------------------------------------

// ContainerUpdateOKBody OK response to ContainerUpdate operation
// swagger:model ContainerUpdateOKBody
type ContainerUpdateOKBody struct {

	// warnings
	// Required: true
	Warnings []string `json:"Warnings"`
}
