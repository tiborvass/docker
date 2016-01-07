// Package client provides a command-line interface for Docker.
//
// Run "docker help SUBCOMMAND" or "docker SUBCOMMAND --help" to see more information on any Docker subcommand, including the full list of options supported for the subcommand.
// See https://docs.docker.com/installation/ for instructions on installing Docker.
package client

import (
	"io"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/registry"
)

// apiClient is an interface that clients that talk with a docker server must implement.
type apiClient interface {
	ClientVersion() string
	ContainerAttach(options types.ContainerAttachOptions) (types.HijackedResponse, error)
	ContainerCommit(options types.ContainerCommitOptions) (types.ContainerCommitResponse, error)
	ContainerCreate(config *container.Config, hostConfig *container.HostConfig, containerName string) (types.ContainerCreateResponse, error)
	ContainerDiff(containerID string) ([]types.ContainerChange, error)
	ContainerExecAttach(execID string, config types.ExecConfig) (types.HijackedResponse, error)
	ContainerExecCreate(config types.ExecConfig) (types.ContainerExecCreateResponse, error)
	ContainerExecInspect(execID string) (types.ContainerExecInspect, error)
	ContainerExecResize(options types.ResizeOptions) error
	ContainerExecStart(execID string, config types.ExecStartCheck) error
	ContainerExport(containerID string) (io.ReadCloser, error)
	ContainerInspect(containerID string) (types.ContainerJSON, error)
	ContainerInspectWithRaw(containerID string, getSize bool) (types.ContainerJSON, []byte, error)
	ContainerKill(containerID, signal string) error
	ContainerList(options types.ContainerListOptions) ([]types.Container, error)
	ContainerLogs(options types.ContainerLogsOptions) (io.ReadCloser, error)
	ContainerPause(containerID string) error
	ContainerRemove(options types.ContainerRemoveOptions) error
	ContainerRename(containerID, newContainerName string) error
	ContainerResize(options types.ResizeOptions) error
	ContainerRestart(containerID string, timeout int) error
	ContainerStatPath(containerID, path string) (types.ContainerPathStat, error)
	ContainerStats(containerID string, stream bool) (io.ReadCloser, error)
	ContainerStart(containerID string) error
	ContainerStop(containerID string, timeout int) error
	ContainerTop(containerID string, arguments []string) (types.ContainerProcessList, error)
	ContainerUnpause(containerID string) error
	ContainerUpdate(containerID string, hostConfig container.HostConfig) error
	ContainerWait(containerID string) (int, error)
	CopyFromContainer(containerID, srcPath string) (io.ReadCloser, types.ContainerPathStat, error)
	CopyToContainer(options types.CopyToContainerOptions) error
	Events(options types.EventsOptions) (io.ReadCloser, error)
	ImageBuild(options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImageCreate(options types.ImageCreateOptions) (io.ReadCloser, error)
	ImageHistory(imageID string) ([]types.ImageHistory, error)
	ImageImport(options types.ImageImportOptions) (io.ReadCloser, error)
	ImageInspectWithRaw(imageID string, getSize bool) (types.ImageInspect, []byte, error)
	ImageList(options types.ImageListOptions) ([]types.Image, error)
	ImageLoad(input io.Reader) (types.ImageLoadResponse, error)
	ImagePull(options types.ImagePullOptions, privilegeFunc client.RequestPrivilegeFunc) (io.ReadCloser, error)
	ImagePush(options types.ImagePushOptions, privilegeFunc client.RequestPrivilegeFunc) (io.ReadCloser, error)
	ImageRemove(options types.ImageRemoveOptions) ([]types.ImageDelete, error)
	ImageSearch(options types.ImageSearchOptions, privilegeFunc client.RequestPrivilegeFunc) ([]registry.SearchResult, error)
	ImageSave(imageIDs []string) (io.ReadCloser, error)
	ImageTag(options types.ImageTagOptions) error
	Info() (types.Info, error)
	NetworkConnect(networkID, containerID string) error
	NetworkCreate(options types.NetworkCreate) (types.NetworkCreateResponse, error)
	NetworkDisconnect(networkID, containerID string) error
	NetworkInspect(networkID string) (types.NetworkResource, error)
	NetworkList(options types.NetworkListOptions) ([]types.NetworkResource, error)
	NetworkRemove(networkID string) error
	RegistryLogin(auth types.AuthConfig) (types.AuthResponse, error)
	ServerVersion() (types.Version, error)
	VolumeCreate(options types.VolumeCreateRequest) (types.Volume, error)
	VolumeInspect(volumeID string) (types.Volume, error)
	VolumeList(filter filters.Args) (types.VolumesListResponse, error)
	VolumeRemove(volumeID string) error
}
