package types

import (
	"bufio"
	"io"
	"net"

	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/api/types/filters"
	"github.com/docker/go-units"
)

// CheckpointCreateOptions holds parameters to create a checkpoint from a container
type CheckpointCreateOptions struct {
	CheckpointID string
	Exit         bool
}

// ContainerAttachOptions holds parameters to attach to a container.
type ContainerAttachOptions struct {
	Stream     bool
	Stdin      bool
	Stdout     bool
	Stderr     bool
	DetachKeys string
}

// ContainerCommitOptions holds parameters to commit changes into a container.
type ContainerCommitOptions struct {
	Reference string
	Comment   string
	Author    string
	Changes   []string
	Pause     bool
	Config    *container.Config
}

// ContainerExecInspect holds information returned by exec inspect.
type ContainerExecInspect struct {
	ExecID      string
	ContainerID string
	Running     bool
	ExitCode    int
}

// ContainerListOptions holds parameters to list containers with.
type ContainerListOptions struct {
	Quiet  bool
	Size   bool
	All    bool
	Latest bool
	Since  string
	Before string
	Limit  int
	Filter filters.Args
}

// ContainerLogsOptions holds parameters to filter logs with.
type ContainerLogsOptions struct {
	ShowStdout bool
	ShowStderr bool
	Since      string
	Timestamps bool
	Follow     bool
	Tail       string
	Details    bool
}

// ContainerRemoveOptions holds parameters to remove containers.
type ContainerRemoveOptions struct {
	RemoveVolumes bool
	RemoveLinks   bool
	Force         bool
}

// ContainerStartOptions holds parameters to start containers.
type ContainerStartOptions struct {
	CheckpointID string
}

// CopyToContainerOptions holds information
// about files to copy into a container
type CopyToContainerOptions struct {
	AllowOverwriteDirWithFile bool
}

// EventsOptions holds parameters to filter events with.
type EventsOptions struct {
	Since   string
	Until   string
	Filters filters.Args
}

// NetworkListOptions holds parameters to filter the list of networks with.
type NetworkListOptions struct {
	Filters filters.Args
}

// HijackedResponse holds connection information for a hijacked request.
type HijackedResponse struct {
	Conn   net.Conn
	Reader *bufio.Reader
}

// Close closes the hijacked connection and reader.
func (h *HijackedResponse) Close() {
	h.Conn.Close()
}

// CloseWriter is an interface that implements structs
// that close input streams to prevent from writing.
type CloseWriter interface {
	CloseWrite() error
}

// CloseWrite closes a readWriter for writing.
func (h *HijackedResponse) CloseWrite() error {
	if conn, ok := h.Conn.(CloseWriter); ok {
		return conn.CloseWrite()
	}
	return nil
}

// ImageBuildOptions holds the information
// necessary to build images.
type ImageBuildOptions struct {
	Tags           []string
	SuppressOutput bool
	RemoteContext  string
	NoCache        bool
	Remove         bool
	ForceRemove    bool
	PullParent     bool
	Isolation      container.Isolation
	CPUSetCPUs     string
	CPUSetMems     string
	CPUShares      int64
	CPUQuota       int64
	CPUPeriod      int64
	Memory         int64
	MemorySwap     int64
	CgroupParent   string
	ShmSize        int64
	Dockerfile     string
	Ulimits        []*units.Ulimit
	BuildArgs      map[string]string
	AuthConfigs    map[string]AuthConfig
	Context        io.Reader
	Labels         map[string]string
	// squash the resulting image's layers to the parent
	// preserves the original image and creates a new one from the parent with all
	// the changes applied to a single layer
	Squash bool
	// CacheFrom specifies images that are used for matching cache. Images
	// specified here do not need to have a valid parent chain to match cache.
	CacheFrom []string
}

// ImageBuildResponse holds information
// returned by a server after building
// an image.
type ImageBuildResponse struct {
	Body   io.ReadCloser
	OSType string
}

// ImageCreateOptions holds information to create images.
type ImageCreateOptions struct {
	RegistryAuth string // RegistryAuth is the base64 encoded credentials for the registry
}

// ImageImportSource holds source information for ImageImport
type ImageImportSource struct {
	Source     io.Reader // Source is the data to send to the server to create this image from (mutually exclusive with SourceName)
	SourceName string    // SourceName is the name of the image to pull (mutually exclusive with Source)
}

// ImageImportOptions holds information to import images from the client host.
type ImageImportOptions struct {
	Tag     string   // Tag is the name to tag this image with. This attribute is deprecated.
	Message string   // Message is the message to tag the image with
	Changes []string // Changes are the raw changes to apply to this image
}

// ImageListOptions holds parameters to filter the list of images with.
type ImageListOptions struct {
	MatchName string
	All       bool
	Filters   filters.Args
}

// ImageLoadResponse returns information to the client about a load process.
type ImageLoadResponse struct {
	// Body must be closed to avoid a resource leak
	Body io.ReadCloser
	JSON bool
}

// ImagePullOptions holds information to pull images.
type ImagePullOptions struct {
	All           bool
	RegistryAuth  string // RegistryAuth is the base64 encoded credentials for the registry
	PrivilegeFunc RequestPrivilegeFunc
}

// RequestPrivilegeFunc is a function interface that
// clients can supply to retry operations after
// getting an authorization error.
// This function returns the registry authentication
// header value in base 64 format, or an error
// if the privilege request fails.
type RequestPrivilegeFunc func() (string, error)

//ImagePushOptions holds information to push images.
type ImagePushOptions ImagePullOptions

// ImageRemoveOptions holds parameters to remove images.
type ImageRemoveOptions struct {
	Force         bool
	PruneChildren bool
}

// ImageSearchOptions holds parameters to search images with.
type ImageSearchOptions struct {
	RegistryAuth  string
	PrivilegeFunc RequestPrivilegeFunc
	Filters       filters.Args
	Limit         int
}

// ResizeOptions holds parameters to resize a tty.
// It can be used to resize container ttys and
// exec process ttys too.
type ResizeOptions struct {
	Height uint
	Width  uint
}

// VersionResponse holds version information for the client and the server
type VersionResponse struct {
	Client *Version
	Server *Version
}

// ServerOK returns true when the client could connect to the docker server
// and parse the information received. It returns false otherwise.
func (v VersionResponse) ServerOK() bool {
	return v.Server != nil
}

// NodeListOptions holds parameters to list nodes with.
type NodeListOptions struct {
	Filter filters.Args
}

// NodeRemoveOptions holds parameters to remove nodes with.
type NodeRemoveOptions struct {
	Force bool
}

// ServiceCreateOptions contains the options to use when creating a service.
type ServiceCreateOptions struct {
	// EncodedRegistryAuth is the encoded registry authorization credentials to
	// use when updating the service.
	//
	// This field follows the format of the X-Registry-Auth header.
	EncodedRegistryAuth string
}

// ServiceCreateResponse contains the information returned to a client
// on the  creation of a new service.
type ServiceCreateResponse struct {
	// ID is the ID of the created service.
	ID string
}

// ServiceUpdateOptions contains the options to be used for updating services.
type ServiceUpdateOptions struct {
	// EncodedRegistryAuth is the encoded registry authorization credentials to
	// use when updating the service.
	//
	// This field follows the format of the X-Registry-Auth header.
	EncodedRegistryAuth string

	// TODO(stevvooe): Consider moving the version parameter of ServiceUpdate
	// into this field. While it does open API users up to racy writes, most
	// users may not need that level of consistency in practice.
}

// ServiceListOptions holds parameters to list  services with.
type ServiceListOptions struct {
	Filter filters.Args
}

// TaskListOptions holds parameters to list  tasks with.
type TaskListOptions struct {
	Filter filters.Args
}

// PluginRemoveOptions holds parameters to remove plugins.
type PluginRemoveOptions struct {
	Force bool
}
