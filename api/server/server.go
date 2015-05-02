package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/gorilla/mux"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/autogen/dockerversion"
	"github.com/tiborvass/docker/builder"
	"github.com/tiborvass/docker/cliconfig"
	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/daemon/networkdriver/bridge"
	"github.com/tiborvass/docker/graph"
	"github.com/tiborvass/docker/pkg/jsonmessage"
	"github.com/tiborvass/docker/pkg/parsers"
	"github.com/tiborvass/docker/pkg/parsers/filters"
	"github.com/tiborvass/docker/pkg/parsers/kernel"
	"github.com/tiborvass/docker/pkg/signal"
	"github.com/tiborvass/docker/pkg/stdcopy"
	"github.com/tiborvass/docker/pkg/streamformatter"
	"github.com/tiborvass/docker/pkg/version"
	"github.com/tiborvass/docker/runconfig"
	"github.com/tiborvass/docker/utils"
)

type ServerConfig struct {
	Logging     bool
	EnableCors  bool
	CorsHeaders string
	Version     string
	SocketGroup string
	Tls         bool
	TlsVerify   bool
	TlsCa       string
	TlsCert     string
	TlsKey      string
}

type Server struct {
	daemon  *daemon.Daemon
	cfg     *ServerConfig
	router  *mux.Router
	start   chan struct{}
	servers []serverCloser
}

func New(cfg *ServerConfig) *Server {
	srv := &Server{
		cfg:   cfg,
		start: make(chan struct{}),
	}
	r := createRouter(srv)
	srv.router = r
	return srv
}

func (s *Server) Close() {
	for _, srv := range s.servers {
		if err := srv.Close(); err != nil {
			logrus.Error(err)
		}
	}
}

func (s *Server) SetDaemon(d *daemon.Daemon) {
	s.daemon = d
}

type serverCloser interface {
	Serve() error
	Close() error
}

// ServeApi loops through all of the protocols sent in to docker and spawns
// off a go routine to setup a serving http.Server for each.
func (s *Server) ServeApi(protoAddrs []string) error {
	var chErrors = make(chan error, len(protoAddrs))

	for _, protoAddr := range protoAddrs {
		protoAddrParts := strings.SplitN(protoAddr, "://", 2)
		if len(protoAddrParts) != 2 {
			return fmt.Errorf("bad format, expected PROTO://ADDR")
		}
		srv, err := s.newServer(protoAddrParts[0], protoAddrParts[1])
		if err != nil {
			return err
		}
		s.servers = append(s.servers, srv)

		go func(proto, addr string) {
			logrus.Infof("Listening for HTTP on %s (%s)", proto, addr)
			if err := srv.Serve(); err != nil && strings.Contains(err.Error(), "use of closed network connection") {
				err = nil
			}
			chErrors <- err
		}(protoAddrParts[0], protoAddrParts[1])
	}

	for i := 0; i < len(protoAddrs); i++ {
		err := <-chErrors
		if err != nil {
			return err
		}
	}

	return nil
}

type HttpServer struct {
	srv *http.Server
	l   net.Listener
}

func (s *HttpServer) Serve() error {
	return s.srv.Serve(s.l)
}
func (s *HttpServer) Close() error {
	return s.l.Close()
}

type HttpApiFunc func(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error

func hijackServer(w http.ResponseWriter) (io.ReadCloser, io.Writer, error) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, nil, err
	}
	// Flush the options to make sure the client sets the raw mode
	conn.Write([]byte{})
	return conn, conn, nil
}

func closeStreams(streams ...interface{}) {
	for _, stream := range streams {
		if tcpc, ok := stream.(interface {
			CloseWrite() error
		}); ok {
			tcpc.CloseWrite()
		} else if closer, ok := stream.(io.Closer); ok {
			closer.Close()
		}
	}
}

// Check to make sure request's Content-Type is application/json
func checkForJson(r *http.Request) error {
	ct := r.Header.Get("Content-Type")

	// No Content-Type header is ok as long as there's no Body
	if ct == "" {
		if r.Body == nil || r.ContentLength == 0 {
			return nil
		}
	}

	// Otherwise it better be json
	if api.MatchesContentType(ct, "application/json") {
		return nil
	}
	return fmt.Errorf("Content-Type specified (%s) must be 'application/json'", ct)
}

//If we don't do this, POST method without Content-type (even with empty body) will fail
func parseForm(r *http.Request) error {
	if r == nil {
		return nil
	}
	if err := r.ParseForm(); err != nil && !strings.HasPrefix(err.Error(), "mime:") {
		return err
	}
	return nil
}

func parseMultipartForm(r *http.Request) error {
	if err := r.ParseMultipartForm(4096); err != nil && !strings.HasPrefix(err.Error(), "mime:") {
		return err
	}
	return nil
}

func httpError(w http.ResponseWriter, err error) {
	if err == nil || w == nil {
		logrus.WithFields(logrus.Fields{"error": err, "writer": w}).Error("unexpected HTTP error handling")
		return
	}
	statusCode := http.StatusInternalServerError
	// FIXME: this is brittle and should not be necessary.
	// If we need to differentiate between different possible error types, we should
	// create appropriate error types with clearly defined meaning.
	errStr := strings.ToLower(err.Error())
	for keyword, status := range map[string]int{
		"not found":             http.StatusNotFound,
		"no such":               http.StatusNotFound,
		"bad parameter":         http.StatusBadRequest,
		"conflict":              http.StatusConflict,
		"impossible":            http.StatusNotAcceptable,
		"wrong login/password":  http.StatusUnauthorized,
		"hasn't been activated": http.StatusForbidden,
	} {
		if strings.Contains(errStr, keyword) {
			statusCode = status
			break
		}
	}

	logrus.WithFields(logrus.Fields{"statusCode": statusCode, "err": err}).Error("HTTP Error")
	http.Error(w, err.Error(), statusCode)
}

// writeJSON writes the value v to the http response stream as json with standard
// json encoding.
func writeJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

func (s *Server) postAuth(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var config *cliconfig.AuthConfig
	err := json.NewDecoder(r.Body).Decode(&config)
	r.Body.Close()
	if err != nil {
		return err
	}
	status, err := s.daemon.RegistryService.Auth(config)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, &types.AuthResponse{
		Status: status,
	})
}

func (s *Server) getVersion(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	w.Header().Set("Content-Type", "application/json")

	v := &types.Version{
		Version:    dockerversion.VERSION,
		ApiVersion: api.APIVERSION,
		GitCommit:  dockerversion.GITCOMMIT,
		GoVersion:  runtime.Version(),
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}
	if kernelVersion, err := kernel.GetKernelVersion(); err == nil {
		v.KernelVersion = kernelVersion.String()
	}

	return writeJSON(w, http.StatusOK, v)
}

func (s *Server) postContainersKill(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}
	if err := parseForm(r); err != nil {
		return err
	}

	var sig uint64
	name := vars["name"]

	// If we have a signal, look at it. Otherwise, do nothing
	if sigStr := vars["signal"]; sigStr != "" {
		// Check if we passed the signal as a number:
		// The largest legal signal is 31, so let's parse on 5 bits
		sig, err := strconv.ParseUint(sigStr, 10, 5)
		if err != nil {
			// The signal is not a number, treat it as a string (either like
			// "KILL" or like "SIGKILL")
			sig = uint64(signal.SignalMap[strings.TrimPrefix(sigStr, "SIG")])
		}

		if sig == 0 {
			return fmt.Errorf("Invalid signal: %s", sigStr)
		}
	}

	if err := s.daemon.ContainerKill(name, sig); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) postContainersPause(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}
	if err := parseForm(r); err != nil {
		return err
	}

	if err := s.daemon.ContainerPause(vars["name"]); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

func (s *Server) postContainersUnpause(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}
	if err := parseForm(r); err != nil {
		return err
	}

	if err := s.daemon.ContainerUnpause(vars["name"]); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

func (s *Server) getContainersExport(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	return s.daemon.ContainerExport(vars["name"], w)
}

func (s *Server) getImagesJSON(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}

	imagesConfig := graph.ImagesConfig{
		Filters: r.Form.Get("filters"),
		// FIXME this parameter could just be a match filter
		Filter: r.Form.Get("filter"),
		All:    boolValue(r, "all"),
	}

	images, err := s.daemon.Repositories().Images(&imagesConfig)
	if err != nil {
		return err
	}

	if version.GreaterThanOrEqualTo("1.7") {
		return writeJSON(w, http.StatusOK, images)
	}

	legacyImages := []types.LegacyImage{}

	for _, image := range images {
		for _, repoTag := range image.RepoTags {
			repo, tag := parsers.ParseRepositoryTag(repoTag)
			legacyImage := types.LegacyImage{
				Repository:  repo,
				Tag:         tag,
				ID:          image.ID,
				Created:     image.Created,
				Size:        image.Size,
				VirtualSize: image.VirtualSize,
			}
			legacyImages = append(legacyImages, legacyImage)
		}
	}

	return writeJSON(w, http.StatusOK, legacyImages)
}

func (s *Server) getInfo(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	w.Header().Set("Content-Type", "application/json")

	info, err := s.daemon.SystemInfo()
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, info)
}

func (s *Server) getEvents(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	var since int64 = -1
	if r.Form.Get("since") != "" {
		s, err := strconv.ParseInt(r.Form.Get("since"), 10, 64)
		if err != nil {
			return err
		}
		since = s
	}

	var until int64 = -1
	if r.Form.Get("until") != "" {
		u, err := strconv.ParseInt(r.Form.Get("until"), 10, 64)
		if err != nil {
			return err
		}
		until = u
	}
	timer := time.NewTimer(0)
	timer.Stop()
	if until > 0 {
		dur := time.Unix(until, 0).Sub(time.Now())
		timer = time.NewTimer(dur)
	}

	ef, err := filters.FromParam(r.Form.Get("filters"))
	if err != nil {
		return err
	}

	isFiltered := func(field string, filter []string) bool {
		if len(filter) == 0 {
			return false
		}
		for _, v := range filter {
			if v == field {
				return false
			}
			if strings.Contains(field, ":") {
				image := strings.Split(field, ":")
				if image[0] == v {
					return false
				}
			}
		}
		return true
	}

	d := s.daemon
	es := d.EventsService
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(utils.NewWriteFlusher(w))

	getContainerId := func(cn string) string {
		c, err := d.Get(cn)
		if err != nil {
			return ""
		}
		return c.ID
	}

	sendEvent := func(ev *jsonmessage.JSONMessage) error {
		//incoming container filter can be name,id or partial id, convert and replace as a full container id
		for i, cn := range ef["container"] {
			ef["container"][i] = getContainerId(cn)
		}

		if isFiltered(ev.Status, ef["event"]) || isFiltered(ev.From, ef["image"]) ||
			isFiltered(ev.ID, ef["container"]) {
			return nil
		}

		return enc.Encode(ev)
	}

	current, l := es.Subscribe()
	defer es.Evict(l)
	for _, ev := range current {
		if ev.Time < since {
			continue
		}
		if err := sendEvent(ev); err != nil {
			return err
		}
	}
	for {
		select {
		case ev := <-l:
			jev, ok := ev.(*jsonmessage.JSONMessage)
			if !ok {
				continue
			}
			if err := sendEvent(jev); err != nil {
				return err
			}
		case <-timer.C:
			return nil
		}
	}
}

func (s *Server) getImagesHistory(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]
	history, err := s.daemon.Repositories().History(name)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, history)
}

func (s *Server) getContainersChanges(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	changes, err := s.daemon.ContainerChanges(vars["name"])
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, changes)
}

func (s *Server) getContainersTop(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if version.LessThan("1.4") {
		return fmt.Errorf("top was improved a lot since 1.3, Please upgrade your docker client.")
	}

	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	if err := parseForm(r); err != nil {
		return err
	}

	procList, err := s.daemon.ContainerTop(vars["name"], r.Form.Get("ps_args"))
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, procList)
}

func (s *Server) getContainersJSON(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}

	config := &daemon.ContainersConfig{
		All:     boolValue(r, "all"),
		Size:    boolValue(r, "size"),
		Since:   r.Form.Get("since"),
		Before:  r.Form.Get("before"),
		Filters: r.Form.Get("filters"),
	}

	if tmpLimit := r.Form.Get("limit"); tmpLimit != "" {
		limit, err := strconv.Atoi(tmpLimit)
		if err != nil {
			return err
		}
		config.Limit = limit
	}

	containers, err := s.daemon.Containers(config)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, containers)
}

func (s *Server) getContainersStats(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	return s.daemon.ContainerStats(vars["name"], utils.NewWriteFlusher(w))
}

func (s *Server) getContainersLogs(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	// Validate args here, because we can't return not StatusOK after job.Run() call
	stdout, stderr := boolValue(r, "stdout"), boolValue(r, "stderr")
	if !(stdout || stderr) {
		return fmt.Errorf("Bad parameters: you must choose at least one stream")
	}

	logsConfig := &daemon.ContainerLogsConfig{
		Follow:     boolValue(r, "follow"),
		Timestamps: boolValue(r, "timestamps"),
		Tail:       r.Form.Get("tail"),
		UseStdout:  stdout,
		UseStderr:  stderr,
		OutStream:  utils.NewWriteFlusher(w),
	}

	if err := s.daemon.ContainerLogs(vars["name"], logsConfig); err != nil {
		fmt.Fprintf(w, "Error running logs job: %s\n", err)
	}

	return nil
}

func (s *Server) postImagesTag(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	repo := r.Form.Get("repo")
	tag := r.Form.Get("tag")
	force := boolValue(r, "force")
	if err := s.daemon.Repositories().Tag(repo, tag, vars["name"], force); err != nil {
		return err
	}
	w.WriteHeader(http.StatusCreated)
	return nil
}

func (s *Server) postCommit(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}

	if err := checkForJson(r); err != nil {
		return err
	}

	cont := r.Form.Get("container")

	pause := boolValue(r, "pause")
	if r.FormValue("pause") == "" && version.GreaterThanOrEqualTo("1.13") {
		pause = true
	}

	c, _, err := runconfig.DecodeContainerConfig(r.Body)
	if err != nil && err != io.EOF { //Do not fail if body is empty.
		return err
	}

	if c == nil {
		c = &runconfig.Config{}
	}

	containerCommitConfig := &daemon.ContainerCommitConfig{
		Pause:   pause,
		Repo:    r.Form.Get("repo"),
		Tag:     r.Form.Get("tag"),
		Author:  r.Form.Get("author"),
		Comment: r.Form.Get("comment"),
		Changes: r.Form["changes"],
		Config:  c,
	}

	imgID, err := builder.Commit(s.daemon, cont, containerCommitConfig)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusCreated, &types.ContainerCommitResponse{
		ID: imgID,
	})
}

// Creates an image from Pull or from Import
func (s *Server) postImagesCreate(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}

	var (
		image = r.Form.Get("fromImage")
		repo  = r.Form.Get("repo")
		tag   = r.Form.Get("tag")
	)
	authEncoded := r.Header.Get("X-Registry-Auth")
	authConfig := &cliconfig.AuthConfig{}
	if authEncoded != "" {
		authJson := base64.NewDecoder(base64.URLEncoding, strings.NewReader(authEncoded))
		if err := json.NewDecoder(authJson).Decode(authConfig); err != nil {
			// for a pull it is not an error if no auth was given
			// to increase compatibility with the existing api it is defaulting to be empty
			authConfig = &cliconfig.AuthConfig{}
		}
	}

	var (
		opErr   error
		useJSON = version.GreaterThan("1.0")
	)

	if useJSON {
		w.Header().Set("Content-Type", "application/json")
	}

	if image != "" { //pull
		if tag == "" {
			image, tag = parsers.ParseRepositoryTag(image)
		}
		metaHeaders := map[string][]string{}
		for k, v := range r.Header {
			if strings.HasPrefix(k, "X-Meta-") {
				metaHeaders[k] = v
			}
		}

		imagePullConfig := &graph.ImagePullConfig{
			Parallel:    version.GreaterThan("1.3"),
			MetaHeaders: metaHeaders,
			AuthConfig:  authConfig,
			OutStream:   utils.NewWriteFlusher(w),
			Json:        useJSON,
		}

		opErr = s.daemon.Repositories().Pull(image, tag, imagePullConfig)
	} else { //import
		if tag == "" {
			repo, tag = parsers.ParseRepositoryTag(repo)
		}

		src := r.Form.Get("fromSrc")
		imageImportConfig := &graph.ImageImportConfig{
			Changes:   r.Form["changes"],
			InConfig:  r.Body,
			OutStream: utils.NewWriteFlusher(w),
			Json:      useJSON,
		}

		newConfig, err := builder.BuildFromConfig(s.daemon, &runconfig.Config{}, imageImportConfig.Changes)
		if err != nil {
			return err
		}
		imageImportConfig.ContainerConfig = newConfig

		opErr = s.daemon.Repositories().Import(src, repo, tag, imageImportConfig)
	}

	if opErr != nil {
		sf := streamformatter.NewStreamFormatter(useJSON)
		return fmt.Errorf(string(sf.FormatError(opErr)))
	}

	return nil
}

func (s *Server) getImagesSearch(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	var (
		config      *cliconfig.AuthConfig
		authEncoded = r.Header.Get("X-Registry-Auth")
		headers     = map[string][]string{}
	)

	if authEncoded != "" {
		authJson := base64.NewDecoder(base64.URLEncoding, strings.NewReader(authEncoded))
		if err := json.NewDecoder(authJson).Decode(&config); err != nil {
			// for a search it is not an error if no auth was given
			// to increase compatibility with the existing api it is defaulting to be empty
			config = &cliconfig.AuthConfig{}
		}
	}
	for k, v := range r.Header {
		if strings.HasPrefix(k, "X-Meta-") {
			headers[k] = v
		}
	}
	query, err := s.daemon.RegistryService.Search(r.Form.Get("term"), config, headers)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(query.Results)
}

func (s *Server) postImagesPush(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	metaHeaders := map[string][]string{}
	for k, v := range r.Header {
		if strings.HasPrefix(k, "X-Meta-") {
			metaHeaders[k] = v
		}
	}
	if err := parseForm(r); err != nil {
		return err
	}
	authConfig := &cliconfig.AuthConfig{}

	authEncoded := r.Header.Get("X-Registry-Auth")
	if authEncoded != "" {
		// the new format is to handle the authConfig as a header
		authJson := base64.NewDecoder(base64.URLEncoding, strings.NewReader(authEncoded))
		if err := json.NewDecoder(authJson).Decode(authConfig); err != nil {
			// to increase compatibility to existing api it is defaulting to be empty
			authConfig = &cliconfig.AuthConfig{}
		}
	} else {
		// the old format is supported for compatibility if there was no authConfig header
		if err := json.NewDecoder(r.Body).Decode(authConfig); err != nil {
			return fmt.Errorf("Bad parameters and missing X-Registry-Auth: %v", err)
		}
	}

	useJSON := version.GreaterThan("1.0")
	name := vars["name"]

	output := utils.NewWriteFlusher(w)
	imagePushConfig := &graph.ImagePushConfig{
		MetaHeaders: metaHeaders,
		AuthConfig:  authConfig,
		Tag:         r.Form.Get("tag"),
		OutStream:   output,
		Json:        useJSON,
	}
	if useJSON {
		w.Header().Set("Content-Type", "application/json")
	}

	if err := s.daemon.Repositories().Push(name, imagePushConfig); err != nil {
		if !output.Flushed() {
			return err
		}
		sf := streamformatter.NewStreamFormatter(useJSON)
		output.Write(sf.FormatError(err))
	}
	return nil

}

func (s *Server) getImagesGet(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}
	if err := parseForm(r); err != nil {
		return err
	}

	useJSON := version.GreaterThan("1.0")
	if useJSON {
		w.Header().Set("Content-Type", "application/x-tar")
	}

	output := utils.NewWriteFlusher(w)
	imageExportConfig := &graph.ImageExportConfig{Outstream: output}
	if name, ok := vars["name"]; ok {
		imageExportConfig.Names = []string{name}
	} else {
		imageExportConfig.Names = r.Form["names"]
	}

	if err := s.daemon.Repositories().ImageExport(imageExportConfig); err != nil {
		if !output.Flushed() {
			return err
		}
		sf := streamformatter.NewStreamFormatter(useJSON)
		output.Write(sf.FormatError(err))
	}
	return nil

}

func (s *Server) postImagesLoad(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	return s.daemon.Repositories().Load(r.Body, w)
}

func (s *Server) postContainersCreate(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return nil
	}
	if err := checkForJson(r); err != nil {
		return err
	}
	var (
		warnings []string
		name     = r.Form.Get("name")
	)

	config, hostConfig, err := runconfig.DecodeContainerConfig(r.Body)
	if err != nil {
		return err
	}

	containerId, warnings, err := s.daemon.ContainerCreate(name, config, hostConfig)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusCreated, &types.ContainerCreateResponse{
		ID:       containerId,
		Warnings: warnings,
	})
}

func (s *Server) postContainersRestart(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	timeout, err := strconv.Atoi(r.Form.Get("t"))
	if err != nil {
		return err
	}

	if err := s.daemon.ContainerRestart(vars["name"], timeout); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

func (s *Server) postContainerRename(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]
	newName := r.Form.Get("name")
	if err := s.daemon.ContainerRename(name, newName); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) deleteContainers(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]
	config := &daemon.ContainerRmConfig{
		ForceRemove:  boolValue(r, "force"),
		RemoveVolume: boolValue(r, "v"),
		RemoveLink:   boolValue(r, "link"),
	}

	if err := s.daemon.ContainerRm(name, config); err != nil {
		// Force a 404 for the empty string
		if strings.Contains(strings.ToLower(err.Error()), "prefix can't be empty") {
			return fmt.Errorf("no such id: \"\"")
		}
		return err
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

func (s *Server) deleteImages(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]
	force := boolValue(r, "force")
	noprune := boolValue(r, "noprune")

	list, err := s.daemon.ImageDelete(name, force, noprune)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, list)
}

func (s *Server) postContainersStart(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	// If contentLength is -1, we can assumed chunked encoding
	// or more technically that the length is unknown
	// https://golang.org/src/pkg/net/http/request.go#L139
	// net/http otherwise seems to swallow any headers related to chunked encoding
	// including r.TransferEncoding
	// allow a nil body for backwards compatibility
	var hostConfig *runconfig.HostConfig
	if r.Body != nil && (r.ContentLength > 0 || r.ContentLength == -1) {
		if err := checkForJson(r); err != nil {
			return err
		}

		c, err := runconfig.DecodeHostConfig(r.Body)
		if err != nil {
			return err
		}

		hostConfig = c
	}

	if err := s.daemon.ContainerStart(vars["name"], hostConfig); err != nil {
		if err.Error() == "Container already started" {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) postContainersStop(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	seconds, err := strconv.Atoi(r.Form.Get("t"))
	if err != nil {
		return err
	}

	if err := s.daemon.ContainerStop(vars["name"], seconds); err != nil {
		if err.Error() == "Container already stopped" {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
		return err
	}
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func (s *Server) postContainersWait(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]
	cont, err := s.daemon.Get(name)
	if err != nil {
		return err
	}

	status, _ := cont.WaitStop(-1 * time.Second)

	return writeJSON(w, http.StatusOK, &types.ContainerWaitResponse{
		StatusCode: status,
	})
}

func (s *Server) postContainersResize(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	height, err := strconv.Atoi(r.Form.Get("h"))
	if err != nil {
		return err
	}
	width, err := strconv.Atoi(r.Form.Get("w"))
	if err != nil {
		return err
	}

	return s.daemon.ContainerResize(vars["name"], height, width)
}

func (s *Server) postContainersAttach(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	cont, err := s.daemon.Get(vars["name"])
	if err != nil {
		return err
	}

	inStream, outStream, err := hijackServer(w)
	if err != nil {
		return err
	}
	defer closeStreams(inStream, outStream)

	var errStream io.Writer

	if _, ok := r.Header["Upgrade"]; ok {
		fmt.Fprintf(outStream, "HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n")
	} else {
		fmt.Fprintf(outStream, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\n")
	}

	if !cont.Config.Tty && version.GreaterThanOrEqualTo("1.6") {
		errStream = stdcopy.NewStdWriter(outStream, stdcopy.Stderr)
		outStream = stdcopy.NewStdWriter(outStream, stdcopy.Stdout)
	} else {
		errStream = outStream
	}
	logs := boolValue(r, "logs")
	stream := boolValue(r, "stream")

	var stdin io.ReadCloser
	var stdout, stderr io.Writer

	if boolValue(r, "stdin") {
		stdin = inStream
	}
	if boolValue(r, "stdout") {
		stdout = outStream
	}
	if boolValue(r, "stderr") {
		stderr = errStream
	}

	if err := cont.AttachWithLogs(stdin, stdout, stderr, logs, stream); err != nil {
		fmt.Fprintf(outStream, "Error attaching: %s\n", err)
	}
	return nil
}

func (s *Server) wsContainersAttach(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}
	cont, err := s.daemon.Get(vars["name"])
	if err != nil {
		return err
	}

	h := websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		logs := r.Form.Get("logs") != ""
		stream := r.Form.Get("stream") != ""

		if err := cont.AttachWithLogs(ws, ws, ws, logs, stream); err != nil {
			logrus.Errorf("Error attaching websocket: %s", err)
		}
	})
	h.ServeHTTP(w, r)

	return nil
}

func (s *Server) getContainersByName(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]

	if version.LessThan("1.12") {
		containerJSONRaw, err := s.daemon.ContainerInspectRaw(name)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, containerJSONRaw)
	}
	containerJSON, err := s.daemon.ContainerInspect(name)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, containerJSON)
}

func (s *Server) getExecByID(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter 'id'")
	}

	eConfig, err := s.daemon.ContainerExecInspect(vars["id"])
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, eConfig)
}

func (s *Server) getImagesByName(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]
	if version.LessThan("1.12") {
		imageInspectRaw, err := s.daemon.Repositories().LookupRaw(name)
		if err != nil {
			return err
		}

		return writeJSON(w, http.StatusOK, imageInspectRaw)
	}

	imageInspect, err := s.daemon.Repositories().Lookup(name)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, imageInspect)
}

func (s *Server) postBuild(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if version.LessThan("1.3") {
		return fmt.Errorf("Multipart upload for build is no longer supported. Please upgrade your docker client.")
	}
	var (
		authEncoded       = r.Header.Get("X-Registry-Auth")
		authConfig        = &cliconfig.AuthConfig{}
		configFileEncoded = r.Header.Get("X-Registry-Config")
		configFile        = &cliconfig.ConfigFile{}
		buildConfig       = builder.NewBuildConfig()
	)

	// This block can be removed when API versions prior to 1.9 are deprecated.
	// Both headers will be parsed and sent along to the daemon, but if a non-empty
	// ConfigFile is present, any value provided as an AuthConfig directly will
	// be overridden. See BuildFile::CmdFrom for details.
	if version.LessThan("1.9") && authEncoded != "" {
		authJson := base64.NewDecoder(base64.URLEncoding, strings.NewReader(authEncoded))
		if err := json.NewDecoder(authJson).Decode(authConfig); err != nil {
			// for a pull it is not an error if no auth was given
			// to increase compatibility with the existing api it is defaulting to be empty
			authConfig = &cliconfig.AuthConfig{}
		}
	}

	if configFileEncoded != "" {
		configFileJson := base64.NewDecoder(base64.URLEncoding, strings.NewReader(configFileEncoded))
		if err := json.NewDecoder(configFileJson).Decode(configFile); err != nil {
			// for a pull it is not an error if no auth was given
			// to increase compatibility with the existing api it is defaulting to be empty
			configFile = &cliconfig.ConfigFile{}
		}
	}

	if version.GreaterThanOrEqualTo("1.8") {
		w.Header().Set("Content-Type", "application/json")
		buildConfig.JSONFormat = true
	}

	if boolValue(r, "forcerm") && version.GreaterThanOrEqualTo("1.12") {
		buildConfig.Remove = true
	} else if r.FormValue("rm") == "" && version.GreaterThanOrEqualTo("1.12") {
		buildConfig.Remove = true
	} else {
		buildConfig.Remove = boolValue(r, "rm")
	}
	if boolValue(r, "pull") && version.GreaterThanOrEqualTo("1.16") {
		buildConfig.Pull = true
	}

	output := utils.NewWriteFlusher(w)
	buildConfig.Stdout = output
	buildConfig.Context = r.Body

	buildConfig.RemoteURL = r.FormValue("remote")
	buildConfig.DockerfileName = r.FormValue("dockerfile")
	buildConfig.RepoName = r.FormValue("t")
	buildConfig.SuppressOutput = boolValue(r, "q")
	buildConfig.NoCache = boolValue(r, "nocache")
	buildConfig.ForceRemove = boolValue(r, "forcerm")
	buildConfig.AuthConfig = authConfig
	buildConfig.ConfigFile = configFile
	buildConfig.MemorySwap = int64Value(r, "memswap")
	buildConfig.Memory = int64Value(r, "memory")
	buildConfig.CpuShares = int64Value(r, "cpushares")
	buildConfig.CpuQuota = int64Value(r, "cpuquota")
	buildConfig.CpuSetCpus = r.FormValue("cpusetcpus")
	buildConfig.CpuSetMems = r.FormValue("cpusetmems")

	// Job cancellation. Note: not all job types support this.
	if closeNotifier, ok := w.(http.CloseNotifier); ok {
		finished := make(chan struct{})
		defer close(finished)
		go func() {
			select {
			case <-finished:
			case <-closeNotifier.CloseNotify():
				logrus.Infof("Client disconnected, cancelling job: build")
				buildConfig.Cancel()
			}
		}()
	}

	if err := builder.Build(s.daemon, buildConfig); err != nil {
		// Do not write the error in the http output if it's still empty.
		// This prevents from writing a 200(OK) when there is an interal error.
		if !output.Flushed() {
			return err
		}
		sf := streamformatter.NewStreamFormatter(version.GreaterThanOrEqualTo("1.8"))
		w.Write(sf.FormatError(err))
	}
	return nil
}

func (s *Server) postContainersCopy(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	if err := checkForJson(r); err != nil {
		return err
	}

	cfg := types.CopyConfig{}
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		return err
	}

	if cfg.Resource == "" {
		return fmt.Errorf("Path cannot be empty")
	}

	data, err := s.daemon.ContainerCopy(vars["name"], cfg.Resource)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such id") {
			w.WriteHeader(http.StatusNotFound)
			return nil
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("Could not find the file %s in container %s", cfg.Resource, vars["name"])
		}
		return err
	}
	defer data.Close()

	w.Header().Set("Content-Type", "application/x-tar")
	if _, err := io.Copy(w, data); err != nil {
		return err
	}

	return nil
}

func (s *Server) postContainerExecCreate(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return nil
	}
	name := vars["name"]

	execConfig := &runconfig.ExecConfig{}
	if err := json.NewDecoder(r.Body).Decode(execConfig); err != nil {
		return err
	}
	execConfig.Container = name

	if len(execConfig.Cmd) == 0 {
		return fmt.Errorf("No exec command specified")
	}

	// Register an instance of Exec in container.
	id, err := s.daemon.ContainerExecCreate(execConfig)
	if err != nil {
		logrus.Errorf("Error setting up exec command in container %s: %s", name, err)
		return err
	}

	return writeJSON(w, http.StatusCreated, &types.ContainerExecCreateResponse{
		ID: id,
	})
}

// TODO(vishh): Refactor the code to avoid having to specify stream config as part of both create and start.
func (s *Server) postContainerExecStart(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return nil
	}
	var (
		execName = vars["name"]
		stdin    io.ReadCloser
		stdout   io.Writer
		stderr   io.Writer
	)

	execStartCheck := &types.ExecStartCheck{}
	if err := json.NewDecoder(r.Body).Decode(execStartCheck); err != nil {
		return err
	}

	if !execStartCheck.Detach {
		// Setting up the streaming http interface.
		inStream, outStream, err := hijackServer(w)
		if err != nil {
			return err
		}
		defer closeStreams(inStream, outStream)

		var errStream io.Writer

		if _, ok := r.Header["Upgrade"]; ok {
			fmt.Fprintf(outStream, "HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n")
		} else {
			fmt.Fprintf(outStream, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\n")
		}

		if !execStartCheck.Tty && version.GreaterThanOrEqualTo("1.6") {
			errStream = stdcopy.NewStdWriter(outStream, stdcopy.Stderr)
			outStream = stdcopy.NewStdWriter(outStream, stdcopy.Stdout)
		} else {
			errStream = outStream
		}
		stdin = inStream
		stdout = outStream
		stderr = errStream
	}
	// Now run the user process in container.

	if err := s.daemon.ContainerExecStart(execName, stdin, stdout, stderr); err != nil {
		logrus.Errorf("Error starting exec command in container %s: %s", execName, err)
		return err
	}
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func (s *Server) postContainerExecResize(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	height, err := strconv.Atoi(r.Form.Get("h"))
	if err != nil {
		return err
	}
	width, err := strconv.Atoi(r.Form.Get("w"))
	if err != nil {
		return err
	}

	return s.daemon.ContainerExecResize(vars["name"], height, width)
}

func (s *Server) optionsHandler(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	w.WriteHeader(http.StatusOK)
	return nil
}
func writeCorsHeaders(w http.ResponseWriter, r *http.Request, corsHeaders string) {
	logrus.Debugf("CORS header is enabled and set to: %s", corsHeaders)
	w.Header().Add("Access-Control-Allow-Origin", corsHeaders)
	w.Header().Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, X-Registry-Auth")
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, OPTIONS")
}

func (s *Server) ping(version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	_, err := w.Write([]byte{'O', 'K'})
	return err
}

func makeHttpHandler(logging bool, localMethod string, localRoute string, handlerFunc HttpApiFunc, corsHeaders string, dockerVersion version.Version) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// log the request
		logrus.Debugf("Calling %s %s", localMethod, localRoute)

		if logging {
			logrus.Infof("%s %s", r.Method, r.RequestURI)
		}

		if strings.Contains(r.Header.Get("User-Agent"), "Docker-Client/") {
			userAgent := strings.Split(r.Header.Get("User-Agent"), "/")
			if len(userAgent) == 2 && !dockerVersion.Equal(version.Version(userAgent[1])) {
				logrus.Debugf("Warning: client and server don't have the same version (client: %s, server: %s)", userAgent[1], dockerVersion)
			}
		}
		version := version.Version(mux.Vars(r)["version"])
		if version == "" {
			version = api.APIVERSION
		}
		if corsHeaders != "" {
			writeCorsHeaders(w, r, corsHeaders)
		}

		if version.GreaterThan(api.APIVERSION) {
			http.Error(w, fmt.Errorf("client and server don't have same version (client API version: %s, server API version: %s)", version, api.APIVERSION).Error(), http.StatusNotFound)
			return
		}

		if err := handlerFunc(version, w, r, mux.Vars(r)); err != nil {
			logrus.Errorf("Handler for %s %s returned error: %s", localMethod, localRoute, err)
			httpError(w, err)
		}
	}
}

// we keep enableCors just for legacy usage, need to be removed in the future
func createRouter(s *Server) *mux.Router {
	r := mux.NewRouter()
	if os.Getenv("DEBUG") != "" {
		ProfilerSetup(r, "/debug/")
	}
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/_ping":                          s.ping,
			"/events":                         s.getEvents,
			"/info":                           s.getInfo,
			"/version":                        s.getVersion,
			"/images/json":                    s.getImagesJSON,
			"/images/search":                  s.getImagesSearch,
			"/images/get":                     s.getImagesGet,
			"/images/{name:.*}/get":           s.getImagesGet,
			"/images/{name:.*}/history":       s.getImagesHistory,
			"/images/{name:.*}/json":          s.getImagesByName,
			"/containers/ps":                  s.getContainersJSON,
			"/containers/json":                s.getContainersJSON,
			"/containers/{name:.*}/export":    s.getContainersExport,
			"/containers/{name:.*}/changes":   s.getContainersChanges,
			"/containers/{name:.*}/json":      s.getContainersByName,
			"/containers/{name:.*}/top":       s.getContainersTop,
			"/containers/{name:.*}/logs":      s.getContainersLogs,
			"/containers/{name:.*}/stats":     s.getContainersStats,
			"/containers/{name:.*}/attach/ws": s.wsContainersAttach,
			"/exec/{id:.*}/json":              s.getExecByID,
		},
		"POST": {
			"/auth":                         s.postAuth,
			"/commit":                       s.postCommit,
			"/build":                        s.postBuild,
			"/images/create":                s.postImagesCreate,
			"/images/load":                  s.postImagesLoad,
			"/images/{name:.*}/push":        s.postImagesPush,
			"/images/{name:.*}/tag":         s.postImagesTag,
			"/containers/create":            s.postContainersCreate,
			"/containers/{name:.*}/kill":    s.postContainersKill,
			"/containers/{name:.*}/pause":   s.postContainersPause,
			"/containers/{name:.*}/unpause": s.postContainersUnpause,
			"/containers/{name:.*}/restart": s.postContainersRestart,
			"/containers/{name:.*}/start":   s.postContainersStart,
			"/containers/{name:.*}/stop":    s.postContainersStop,
			"/containers/{name:.*}/wait":    s.postContainersWait,
			"/containers/{name:.*}/resize":  s.postContainersResize,
			"/containers/{name:.*}/attach":  s.postContainersAttach,
			"/containers/{name:.*}/copy":    s.postContainersCopy,
			"/containers/{name:.*}/exec":    s.postContainerExecCreate,
			"/exec/{name:.*}/start":         s.postContainerExecStart,
			"/exec/{name:.*}/resize":        s.postContainerExecResize,
			"/containers/{name:.*}/rename":  s.postContainerRename,
		},
		"DELETE": {
			"/containers/{name:.*}": s.deleteContainers,
			"/images/{name:.*}":     s.deleteImages,
		},
		"OPTIONS": {
			"": s.optionsHandler,
		},
	}

	// If "api-cors-header" is not given, but "api-enable-cors" is true, we set cors to "*"
	// otherwise, all head values will be passed to HTTP handler
	corsHeaders := s.cfg.CorsHeaders
	if corsHeaders == "" && s.cfg.EnableCors {
		corsHeaders = "*"
	}

	for method, routes := range m {
		for route, fct := range routes {
			logrus.Debugf("Registering %s, %s", method, route)
			// NOTE: scope issue, make sure the variables are local and won't be changed
			localRoute := route
			localFct := fct
			localMethod := method

			// build the handler function
			f := makeHttpHandler(s.cfg.Logging, localMethod, localRoute, localFct, corsHeaders, version.Version(s.cfg.Version))

			// add the new route
			if localRoute == "" {
				r.Methods(localMethod).HandlerFunc(f)
			} else {
				r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(f)
				r.Path(localRoute).Methods(localMethod).HandlerFunc(f)
			}
		}
	}

	return r
}

func allocateDaemonPort(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	intPort, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	var hostIPs []net.IP
	if parsedIP := net.ParseIP(host); parsedIP != nil {
		hostIPs = append(hostIPs, parsedIP)
	} else if hostIPs, err = net.LookupIP(host); err != nil {
		return fmt.Errorf("failed to lookup %s address in host specification", host)
	}

	for _, hostIP := range hostIPs {
		if _, err := bridge.RequestPort(hostIP, "tcp", intPort); err != nil {
			return fmt.Errorf("failed to allocate daemon listening port %d (err: %v)", intPort, err)
		}
	}
	return nil
}
