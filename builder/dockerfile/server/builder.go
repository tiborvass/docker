package server

import (
	"io"
	"os"

	"github.com/docker/docker/builder"
	"github.com/docker/docker/pkg/ulimit"
)

type Config struct {
	//dockerfile.BuilderConfig
	DockerfileName string

	// TODO: possibly not needed if appropriate context is passed (or deduced)
	RemoteURL string

	// TODO: not needed if it's only to tag the image: responsibility of the caller
	RepoName string

	SuppressOutput bool
	NoCache        bool
	Remove         bool
	ForceRemove    bool
	Pull           bool

	// resource constraints
	// TODO: factor out to be reused with Run ?

	Memory       int64
	MemorySwap   int64
	CPUShares    int64
	CPUPeriod    int64
	CPUQuota     int64
	CPUSetCpus   string
	CPUSetMems   string
	CgroupParent string
	Ulimits      []*ulimit.Ulimit
}

type Builder struct {
	*Config
	Stdout io.Writer
	Stderr io.Writer
}

func NewBuilder(config *Config) *Builder {
	if config == nil {
		config = new(Config)
	}
	return &Builder{
		Config: config,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (b Builder) Build(context builder.Context) (builder.ImageID, error) {

}
