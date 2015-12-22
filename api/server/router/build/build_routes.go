package build

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api/server/httputils"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/builder"
	"github.com/tiborvass/docker/builder/dockerfile"
	"github.com/tiborvass/docker/daemon/daemonbuilder"
	"github.com/tiborvass/docker/pkg/archive"
	"github.com/tiborvass/docker/pkg/chrootarchive"
	"github.com/tiborvass/docker/pkg/ioutils"
	"github.com/tiborvass/docker/pkg/progress"
	"github.com/tiborvass/docker/pkg/streamformatter"
	"github.com/tiborvass/docker/pkg/ulimit"
	"github.com/tiborvass/docker/reference"
	"github.com/tiborvass/docker/utils"
	"golang.org/x/net/context"
)

// sanitizeRepoAndTags parses the raw "t" parameter received from the client
// to a slice of repoAndTag.
// It also validates each repoName and tag.
func sanitizeRepoAndTags(names []string) ([]reference.Named, error) {
	var (
		repoAndTags []reference.Named
		// This map is used for deduplicating the "-t" parameter.
		uniqNames = make(map[string]struct{})
	)
	for _, repo := range names {
		if repo == "" {
			continue
		}

		ref, err := reference.ParseNamed(repo)
		if err != nil {
			return nil, err
		}

		ref = reference.WithDefaultTag(ref)

		if _, isCanonical := ref.(reference.Canonical); isCanonical {
			return nil, errors.New("build tag cannot contain a digest")
		}

		if _, isTagged := ref.(reference.NamedTagged); !isTagged {
			ref, err = reference.WithTag(ref, reference.DefaultTag)
		}

		nameWithTag := ref.String()

		if _, exists := uniqNames[nameWithTag]; !exists {
			uniqNames[nameWithTag] = struct{}{}
			repoAndTags = append(repoAndTags, ref)
		}
	}
	return repoAndTags, nil
}

func (br *buildRouter) postBuild(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var (
		authConfigs        = map[string]types.AuthConfig{}
		authConfigsEncoded = r.Header.Get("X-Registry-Config")
		buildConfig        = &dockerfile.Config{}
		notVerboseBuffer   = bytes.NewBuffer(nil)
	)

	if authConfigsEncoded != "" {
		authConfigsJSON := base64.NewDecoder(base64.URLEncoding, strings.NewReader(authConfigsEncoded))
		if err := json.NewDecoder(authConfigsJSON).Decode(&authConfigs); err != nil {
			// for a pull it is not an error if no auth was given
			// to increase compatibility with the existing api it is defaulting
			// to be empty.
		}
	}

	w.Header().Set("Content-Type", "application/json")

	version := httputils.VersionFromContext(ctx)
	output := ioutils.NewWriteFlusher(w)
	defer output.Close()
	sf := streamformatter.NewJSONStreamFormatter()
	errf := func(err error) error {
		if !buildConfig.Verbose && notVerboseBuffer.Len() > 0 {
			output.Write(notVerboseBuffer.Bytes())
		}
		// Do not write the error in the http output if it's still empty.
		// This prevents from writing a 200(OK) when there is an internal error.
		if !output.Flushed() {
			return err
		}
		_, err = w.Write(sf.FormatError(errors.New(utils.GetErrorMessage(err))))
		if err != nil {
			logrus.Warnf("could not write error response: %v", err)
		}
		return nil
	}

	if httputils.BoolValue(r, "forcerm") && version.GreaterThanOrEqualTo("1.12") {
		buildConfig.Remove = true
	} else if r.FormValue("rm") == "" && version.GreaterThanOrEqualTo("1.12") {
		buildConfig.Remove = true
	} else {
		buildConfig.Remove = httputils.BoolValue(r, "rm")
	}
	if httputils.BoolValue(r, "pull") && version.GreaterThanOrEqualTo("1.16") {
		buildConfig.Pull = true
	}

	repoAndTags, err := sanitizeRepoAndTags(r.Form["t"])
	if err != nil {
		return errf(err)
	}

	buildConfig.DockerfileName = r.FormValue("dockerfile")
	buildConfig.Verbose = !httputils.BoolValue(r, "q")
	buildConfig.UseCache = !httputils.BoolValue(r, "nocache")
	buildConfig.ForceRemove = httputils.BoolValue(r, "forcerm")
	buildConfig.MemorySwap = httputils.Int64ValueOrZero(r, "memswap")
	buildConfig.Memory = httputils.Int64ValueOrZero(r, "memory")
	buildConfig.CPUShares = httputils.Int64ValueOrZero(r, "cpushares")
	buildConfig.CPUPeriod = httputils.Int64ValueOrZero(r, "cpuperiod")
	buildConfig.CPUQuota = httputils.Int64ValueOrZero(r, "cpuquota")
	buildConfig.CPUSetCpus = r.FormValue("cpusetcpus")
	buildConfig.CPUSetMems = r.FormValue("cpusetmems")
	buildConfig.CgroupParent = r.FormValue("cgroupparent")

	if r.Form.Get("shmsize") != "" {
		shmSize, err := strconv.ParseInt(r.Form.Get("shmsize"), 10, 64)
		if err != nil {
			return errf(err)
		}
		buildConfig.ShmSize = &shmSize
	}

	if i := container.IsolationLevel(r.FormValue("isolation")); i != "" {
		if !container.IsolationLevel.IsValid(i) {
			return errf(fmt.Errorf("Unsupported isolation: %q", i))
		}
		buildConfig.Isolation = i
	}

	var buildUlimits = []*ulimit.Ulimit{}
	ulimitsJSON := r.FormValue("ulimits")
	if ulimitsJSON != "" {
		if err := json.NewDecoder(strings.NewReader(ulimitsJSON)).Decode(&buildUlimits); err != nil {
			return errf(err)
		}
		buildConfig.Ulimits = buildUlimits
	}

	var buildArgs = map[string]string{}
	buildArgsJSON := r.FormValue("buildargs")
	if buildArgsJSON != "" {
		if err := json.NewDecoder(strings.NewReader(buildArgsJSON)).Decode(&buildArgs); err != nil {
			return errf(err)
		}
		buildConfig.BuildArgs = buildArgs
	}

	remoteURL := r.FormValue("remote")

	// Currently, only used if context is from a remote url.
	// Look at code in DetectContextFromRemoteURL for more information.
	createProgressReader := func(in io.ReadCloser) io.ReadCloser {
		progressOutput := sf.NewProgressOutput(output, true)
		if !buildConfig.Verbose {
			progressOutput = sf.NewProgressOutput(notVerboseBuffer, true)
		}
		return progress.NewProgressReader(in, progressOutput, r.ContentLength, "Downloading context", remoteURL)
	}

	var (
		context        builder.ModifiableContext
		dockerfileName string
	)
	context, dockerfileName, err = daemonbuilder.DetectContextFromRemoteURL(r.Body, remoteURL, createProgressReader)
	if err != nil {
		return errf(err)
	}
	defer func() {
		if err := context.Close(); err != nil {
			logrus.Debugf("[BUILDER] failed to remove temporary context: %v", err)
		}
	}()

	uidMaps, gidMaps := br.backend.GetUIDGIDMaps()
	defaultArchiver := &archive.Archiver{
		Untar:   chrootarchive.Untar,
		UIDMaps: uidMaps,
		GIDMaps: gidMaps,
	}
	docker := &daemonbuilder.Docker{
		Daemon:      br.backend,
		OutOld:      output,
		AuthConfigs: authConfigs,
		Archiver:    defaultArchiver,
	}
	if !buildConfig.Verbose {
		docker.OutOld = notVerboseBuffer
	}

	b, err := dockerfile.NewBuilder(buildConfig, docker, builder.DockerIgnoreContext{ModifiableContext: context}, nil)
	if err != nil {
		return errf(err)
	}
	b.Stdout = &streamformatter.StdoutFormatter{Writer: output, StreamFormatter: sf}
	b.Stderr = &streamformatter.StderrFormatter{Writer: output, StreamFormatter: sf}
	if !buildConfig.Verbose {
		b.Stdout = &streamformatter.StdoutFormatter{Writer: notVerboseBuffer, StreamFormatter: sf}
		b.Stderr = &streamformatter.StderrFormatter{Writer: notVerboseBuffer, StreamFormatter: sf}
	}

	if closeNotifier, ok := w.(http.CloseNotifier); ok {
		finished := make(chan struct{})
		defer close(finished)
		go func() {
			select {
			case <-finished:
			case <-closeNotifier.CloseNotify():
				logrus.Infof("Client disconnected, cancelling job: build")
				b.Cancel()
			}
		}()
	}

	if len(dockerfileName) > 0 {
		b.DockerfileName = dockerfileName
	}

	imgID, err := b.Build()
	if err != nil {
		return errf(err)
	}

	for _, rt := range repoAndTags {
		if err := br.backend.TagImage(rt, imgID); err != nil {
			return errf(err)
		}
	}

	// Everything worked so if -q was provided the output from the daemon
	// should be just the image ID and we'll print that to stdout.
	if !buildConfig.Verbose {
		stdout := &streamformatter.StdoutFormatter{Writer: output, StreamFormatter: sf}
		fmt.Fprintf(stdout, "%s\n", string(imgID))
	}

	return nil
}
