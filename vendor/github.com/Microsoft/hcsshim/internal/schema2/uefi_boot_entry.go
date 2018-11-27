/*
 * HCS API
 *
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * API version: 2.1
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package hcsschema

type UefiBootEntry struct {

	DeviceType string `json:"DeviceType,omitempty"`

	DevicePath string `json:"DevicePath,omitempty"`

	DiskNumber int32 `json:"DiskNumber,omitempty"`

	OptionalData string `json:"OptionalData,omitempty"`

	VmbFsRootPath string `json:"VmbFsRootPath,omitempty"`
}
