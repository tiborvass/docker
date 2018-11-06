/*
 * HCS API
 *
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * API version: 2.1
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package hcsschema

import (
	"time"
)

//  Information about a process running in a container
type ProcessDetails struct {

	ProcessId int32 `json:"ProcessId,omitempty"`

	ImageName string `json:"ImageName,omitempty"`

	CreateTimestamp time.Time `json:"CreateTimestamp,omitempty"`

	UserTime100ns int32 `json:"UserTime100ns,omitempty"`

	KernelTime100ns int32 `json:"KernelTime100ns,omitempty"`

	MemoryCommitBytes int32 `json:"MemoryCommitBytes,omitempty"`

	MemoryWorkingSetPrivateBytes int32 `json:"MemoryWorkingSetPrivateBytes,omitempty"`

	MemoryWorkingSetSharedBytes int32 `json:"MemoryWorkingSetSharedBytes,omitempty"`
}
