package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/containerd/containerd"
	"github.com/docker/distribution/reference"
	"github.com/gogo/googleapis/google/rpc"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes/any"
	apitypes "github.com/moby/buildkit/api/types"
	"github.com/moby/buildkit/cache"
	cacheutil "github.com/moby/buildkit/cache/util"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/executor"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/frontend"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	pb "github.com/moby/buildkit/frontend/gateway/pb"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/solver"
	"github.com/moby/buildkit/solver/errdefs"
	opspb "github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/apicaps"
	"github.com/moby/buildkit/util/grpcerrors"
	"github.com/moby/buildkit/util/stack"
	"github.com/moby/buildkit/util/tracing"
	"github.com/moby/buildkit/worker"
	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const (
	keySource = "source"
	keyDevel  = "gateway-devel"
)

func NewGatewayFrontend(w frontend.WorkerInfos) frontend.Frontend {
	return &gatewayFrontend{
		workers: w,
	}
}

type gatewayFrontend struct {
	workers frontend.WorkerInfos
}

func filterPrefix(opts map[string]string, pfx string) map[string]string {
	m := map[string]string{}
	for k, v := range opts {
		if strings.HasPrefix(k, pfx) {
			m[strings.TrimPrefix(k, pfx)] = v
		}
	}
	return m
}

func (gf *gatewayFrontend) Solve(ctx context.Context, llbBridge frontend.FrontendLLBBridge, opts map[string]string, inputs map[string]*opspb.Definition, sid string, sm *session.Manager) (*frontend.Result, error) {
	source, ok := opts[keySource]
	if !ok {
		return nil, errors.Errorf("no source specified for gateway")
	}

	_, isDevel := opts[keyDevel]
	var img specs.Image
	var mfstDigest digest.Digest
	var rootFS cache.MutableRef
	var readonly bool // TODO: try to switch to read-only by default.

	if isDevel {
		devRes, err := llbBridge.Solve(ctx,
			frontend.SolveRequest{
				Frontend:       source,
				FrontendOpt:    filterPrefix(opts, "gateway-"),
				FrontendInputs: inputs,
			}, "gateway:"+sid)
		if err != nil {
			return nil, err
		}
		defer func() {
			devRes.EachRef(func(ref solver.ResultProxy) error {
				return ref.Release(context.TODO())
			})
		}()
		if devRes.Ref == nil {
			return nil, errors.Errorf("development gateway didn't return default result")
		}
		res, err := devRes.Ref.Result(ctx)
		if err != nil {
			return nil, err
		}
		workerRef, ok := res.Sys().(*worker.WorkerRef)
		if !ok {
			return nil, errors.Errorf("invalid ref: %T", res.Sys())
		}

		rootFS, err = workerRef.Worker.CacheManager().New(ctx, workerRef.ImmutableRef, session.NewGroup(sid))
		if err != nil {
			return nil, err
		}
		defer rootFS.Release(context.TODO())
		config, ok := devRes.Metadata[exptypes.ExporterImageConfigKey]
		if ok {
			if err := json.Unmarshal(config, &img); err != nil {
				return nil, err
			}
		}
	} else {
		sourceRef, err := reference.ParseNormalizedNamed(source)
		if err != nil {
			return nil, err
		}

		dgst, config, err := llbBridge.ResolveImageConfig(ctx, reference.TagNameOnly(sourceRef).String(), llb.ResolveImageConfigOpt{})
		if err != nil {
			return nil, err
		}
		mfstDigest = dgst

		if err := json.Unmarshal(config, &img); err != nil {
			return nil, err
		}

		if dgst != "" {
			sourceRef, err = reference.WithDigest(sourceRef, dgst)
			if err != nil {
				return nil, err
			}
		}

		src := llb.Image(sourceRef.String(), &markTypeFrontend{})

		def, err := src.Marshal(ctx)
		if err != nil {
			return nil, err
		}

		res, err := llbBridge.Solve(ctx, frontend.SolveRequest{
			Definition: def.ToPB(),
		}, sid)
		if err != nil {
			return nil, err
		}
		defer func() {
			res.EachRef(func(ref solver.ResultProxy) error {
				return ref.Release(context.TODO())
			})
		}()
		if res.Ref == nil {
			return nil, errors.Errorf("gateway source didn't return default result")

		}
		r, err := res.Ref.Result(ctx)
		if err != nil {
			return nil, err
		}
		workerRef, ok := r.Sys().(*worker.WorkerRef)
		if !ok {
			return nil, errors.Errorf("invalid ref: %T", r.Sys())
		}
		rootFS, err = workerRef.Worker.CacheManager().New(ctx, workerRef.ImmutableRef, session.NewGroup(sid))
		if err != nil {
			return nil, err
		}
		defer rootFS.Release(context.TODO())
	}

	args := []string{"/run"}
	env := []string{}
	cwd := "/"
	if img.Config.Env != nil {
		env = img.Config.Env
	}
	if img.Config.Entrypoint != nil {
		args = img.Config.Entrypoint
	}
	if img.Config.WorkingDir != "" {
		cwd = img.Config.WorkingDir
	}
	i := 0
	for k, v := range opts {
		env = append(env, fmt.Sprintf("BUILDKIT_FRONTEND_OPT_%d", i)+"="+k+"="+v)
		i++
	}

	env = append(env, "BUILDKIT_SESSION_ID="+sid)

	dt, err := json.Marshal(gf.workers.WorkerInfos())
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal workers array")
	}
	env = append(env, "BUILDKIT_WORKERS="+string(dt))

	env = append(env, "BUILDKIT_EXPORTEDPRODUCT="+apicaps.ExportedProduct)

	meta := executor.Meta{
		Env:            env,
		Args:           args,
		Cwd:            cwd,
		ReadonlyRootFS: readonly,
	}

	if v, ok := img.Config.Labels["moby.buildkit.frontend.network.none"]; ok {
		if ok, _ := strconv.ParseBool(v); ok {
			meta.NetMode = opspb.NetMode_NONE
		}
	}

	curCaps := getCaps(img.Config.Labels["moby.buildkit.frontend.caps"])
	addCapsForKnownFrontends(curCaps, mfstDigest)
	reqCaps := getCaps(opts["frontend.caps"])
	if len(inputs) > 0 {
		reqCaps["moby.buildkit.frontend.inputs"] = struct{}{}
	}

	for c := range reqCaps {
		if _, ok := curCaps[c]; !ok {
			return nil, stack.Enable(grpcerrors.WrapCode(errdefs.NewUnsupportedFrontendCapError(c), codes.Unimplemented))
		}
	}

	lbf, ctx, err := serveLLBBridgeForwarder(ctx, llbBridge, gf.workers, inputs, sid, sm)
	defer lbf.conn.Close() //nolint
	if err != nil {
		return nil, err
	}
	defer lbf.Discard()

	err = llbBridge.Run(ctx, "", mountWithSession(rootFS, session.NewGroup(sid)), nil, executor.ProcessInfo{Meta: meta, Stdin: lbf.Stdin, Stdout: lbf.Stdout, Stderr: os.Stderr}, nil)

	if err != nil {
		if errors.Is(err, context.Canceled) && lbf.isErrServerClosed {
			err = errors.Errorf("frontend grpc server closed unexpectedly")
		}
		// An existing error (set via Return rpc) takes
		// precedence over this error, which in turn takes
		// precedence over a success reported via Return.
		lbf.mu.Lock()
		if lbf.err == nil {
			lbf.result = nil
			lbf.err = err
		}
		lbf.mu.Unlock()
	}

	return lbf.Result()
}

func (lbf *llbBridgeForwarder) Discard() {
	lbf.mu.Lock()
	defer lbf.mu.Unlock()
	for id, r := range lbf.refs {
		if lbf.err == nil && lbf.result != nil {
			keep := false
			lbf.result.EachRef(func(r2 solver.ResultProxy) error {
				if r == r2 {
					keep = true
				}
				return nil
			})
			if keep {
				continue
			}
		}
		r.Release(context.TODO())
		delete(lbf.refs, id)
	}
}

func (lbf *llbBridgeForwarder) Done() <-chan struct{} {
	return lbf.doneCh
}

func (lbf *llbBridgeForwarder) setResult(r *frontend.Result, err error) (*pb.ReturnResponse, error) {
	lbf.mu.Lock()
	defer lbf.mu.Unlock()

	if (r == nil) == (err == nil) {
		return nil, errors.New("gateway return must be either result or err")
	}

	if lbf.result != nil || lbf.err != nil {
		return nil, errors.New("gateway result is already set")
	}

	lbf.result = r
	lbf.err = err
	close(lbf.doneCh)
	return &pb.ReturnResponse{}, nil
}

func (lbf *llbBridgeForwarder) Result() (*frontend.Result, error) {
	lbf.mu.Lock()
	defer lbf.mu.Unlock()

	if lbf.result == nil && lbf.err == nil {
		return nil, errors.New("no result for incomplete build")
	}

	if lbf.err != nil {
		return nil, lbf.err
	}

	return lbf.result, nil
}

func NewBridgeForwarder(ctx context.Context, llbBridge frontend.FrontendLLBBridge, workers frontend.WorkerInfos, inputs map[string]*opspb.Definition, sid string, sm *session.Manager) LLBBridgeForwarder {
	return newBridgeForwarder(ctx, llbBridge, workers, inputs, sid, sm)
}

func newBridgeForwarder(ctx context.Context, llbBridge frontend.FrontendLLBBridge, workers frontend.WorkerInfos, inputs map[string]*opspb.Definition, sid string, sm *session.Manager) *llbBridgeForwarder {
	lbf := &llbBridgeForwarder{
		callCtx:   ctx,
		llbBridge: llbBridge,
		refs:      map[string]solver.ResultProxy{},
		doneCh:    make(chan struct{}),
		pipe:      newPipe(),
		workers:   workers,
		inputs:    inputs,
		sid:       sid,
		sm:        sm,
		ctrs:      map[string]gwclient.Container{},
	}
	return lbf
}

func serveLLBBridgeForwarder(ctx context.Context, llbBridge frontend.FrontendLLBBridge, workers frontend.WorkerInfos, inputs map[string]*opspb.Definition, sid string, sm *session.Manager) (*llbBridgeForwarder, context.Context, error) {
	ctx, cancel := context.WithCancel(ctx)
	lbf := newBridgeForwarder(ctx, llbBridge, workers, inputs, sid, sm)
	server := grpc.NewServer(grpc.UnaryInterceptor(grpcerrors.UnaryServerInterceptor), grpc.StreamInterceptor(grpcerrors.StreamServerInterceptor))
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())
	pb.RegisterLLBBridgeServer(server, lbf)

	go func() {
		serve(ctx, server, lbf.conn)
		select {
		case <-ctx.Done():
		default:
			lbf.isErrServerClosed = true
		}
		cancel()
	}()

	return lbf, ctx, nil
}

type pipe struct {
	Stdin  io.ReadCloser
	Stdout io.WriteCloser
	conn   net.Conn
}

func newPipe() *pipe {
	pr1, pw1, _ := os.Pipe()
	pr2, pw2, _ := os.Pipe()
	return &pipe{
		Stdin:  pr1,
		Stdout: pw2,
		conn: &conn{
			Reader: pr2,
			Writer: pw1,
			Closer: pw2,
		},
	}
}

type conn struct {
	io.Reader
	io.Writer
	io.Closer
}

func (s *conn) LocalAddr() net.Addr {
	return dummyAddr{}
}
func (s *conn) RemoteAddr() net.Addr {
	return dummyAddr{}
}
func (s *conn) SetDeadline(t time.Time) error {
	return nil
}
func (s *conn) SetReadDeadline(t time.Time) error {
	return nil
}
func (s *conn) SetWriteDeadline(t time.Time) error {
	return nil
}

type dummyAddr struct {
}

func (d dummyAddr) Network() string {
	return "pipe"
}

func (d dummyAddr) String() string {
	return "localhost"
}

type LLBBridgeForwarder interface {
	pb.LLBBridgeServer
	Done() <-chan struct{}
	Result() (*frontend.Result, error)
	Discard()
}

type llbBridgeForwarder struct {
	mu        sync.Mutex
	callCtx   context.Context
	llbBridge frontend.FrontendLLBBridge
	refs      map[string]solver.ResultProxy
	// lastRef      solver.CachedResult
	// lastRefs     map[string]solver.CachedResult
	// err          error
	doneCh            chan struct{} // closed when result or err become valid through a call to a Return
	result            *frontend.Result
	err               error
	workers           frontend.WorkerInfos
	inputs            map[string]*opspb.Definition
	isErrServerClosed bool
	sid               string
	sm                *session.Manager
	*pipe
	ctrs   map[string]gwclient.Container
	ctrsMu sync.Mutex
}

func (lbf *llbBridgeForwarder) ResolveImageConfig(ctx context.Context, req *pb.ResolveImageConfigRequest) (*pb.ResolveImageConfigResponse, error) {
	ctx = tracing.ContextWithSpanFromContext(ctx, lbf.callCtx)
	var platform *specs.Platform
	if p := req.Platform; p != nil {
		platform = &specs.Platform{
			OS:           p.OS,
			Architecture: p.Architecture,
			Variant:      p.Variant,
			OSVersion:    p.OSVersion,
			OSFeatures:   p.OSFeatures,
		}
	}
	dgst, dt, err := lbf.llbBridge.ResolveImageConfig(ctx, req.Ref, llb.ResolveImageConfigOpt{
		Platform:    platform,
		ResolveMode: req.ResolveMode,
		LogName:     req.LogName,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ResolveImageConfigResponse{
		Digest: dgst,
		Config: dt,
	}, nil
}

func translateLegacySolveRequest(req *pb.SolveRequest) error {
	// translates ImportCacheRefs to new CacheImports (v0.4.0)
	for _, legacyImportRef := range req.ImportCacheRefsDeprecated {
		im := &pb.CacheOptionsEntry{
			Type:  "registry",
			Attrs: map[string]string{"ref": legacyImportRef},
		}
		// FIXME(AkihiroSuda): skip append if already exists
		req.CacheImports = append(req.CacheImports, im)
	}
	req.ImportCacheRefsDeprecated = nil
	return nil
}

func (lbf *llbBridgeForwarder) Solve(ctx context.Context, req *pb.SolveRequest) (*pb.SolveResponse, error) {
	if err := translateLegacySolveRequest(req); err != nil {
		return nil, err
	}
	var cacheImports []frontend.CacheOptionsEntry
	for _, e := range req.CacheImports {
		cacheImports = append(cacheImports, frontend.CacheOptionsEntry{
			Type:  e.Type,
			Attrs: e.Attrs,
		})
	}

	ctx = tracing.ContextWithSpanFromContext(ctx, lbf.callCtx)
	res, err := lbf.llbBridge.Solve(ctx, frontend.SolveRequest{
		Definition:     req.Definition,
		Frontend:       req.Frontend,
		FrontendOpt:    req.FrontendOpt,
		FrontendInputs: req.FrontendInputs,
		CacheImports:   cacheImports,
	}, lbf.sid)
	if err != nil {
		return nil, err
	}

	if len(res.Refs) > 0 && !req.AllowResultReturn {
		// this should never happen because old client shouldn't make a map request
		return nil, errors.Errorf("solve did not return default result")
	}

	pbRes := &pb.Result{
		Metadata: res.Metadata,
	}
	var defaultID string

	lbf.mu.Lock()
	if res.Refs != nil {
		ids := make(map[string]string, len(res.Refs))
		defs := make(map[string]*opspb.Definition, len(res.Refs))
		for k, ref := range res.Refs {
			id := identity.NewID()
			if ref == nil {
				id = ""
			} else {
				lbf.refs[id] = ref
			}
			ids[k] = id
			defs[k] = ref.Definition()
		}

		if req.AllowResultArrayRef {
			refMap := make(map[string]*pb.Ref, len(res.Refs))
			for k, id := range ids {
				refMap[k] = &pb.Ref{Id: id, Def: defs[k]}
			}
			pbRes.Result = &pb.Result_Refs{Refs: &pb.RefMap{Refs: refMap}}
		} else {
			pbRes.Result = &pb.Result_RefsDeprecated{RefsDeprecated: &pb.RefMapDeprecated{Refs: ids}}
		}
	} else {
		ref := res.Ref
		id := identity.NewID()

		var def *opspb.Definition
		if ref == nil {
			id = ""
		} else {
			def = ref.Definition()
			lbf.refs[id] = ref
		}
		defaultID = id

		if req.AllowResultArrayRef {
			pbRes.Result = &pb.Result_Ref{Ref: &pb.Ref{Id: id, Def: def}}
		} else {
			pbRes.Result = &pb.Result_RefDeprecated{RefDeprecated: id}
		}
	}
	lbf.mu.Unlock()

	// compatibility mode for older clients
	if req.Final {
		exp := map[string][]byte{}
		if err := json.Unmarshal(req.ExporterAttr, &exp); err != nil {
			return nil, err
		}

		for k, v := range res.Metadata {
			exp[k] = v
		}

		lbf.mu.Lock()
		lbf.result = &frontend.Result{
			Ref:      lbf.refs[defaultID],
			Metadata: exp,
		}
		lbf.mu.Unlock()
	}

	resp := &pb.SolveResponse{
		Result: pbRes,
	}

	if !req.AllowResultReturn {
		resp.Ref = defaultID
	}

	return resp, nil
}
func (lbf *llbBridgeForwarder) ReadFile(ctx context.Context, req *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	ctx = tracing.ContextWithSpanFromContext(ctx, lbf.callCtx)
	lbf.mu.Lock()
	ref, ok := lbf.refs[req.Ref]
	lbf.mu.Unlock()
	if !ok {
		return nil, errors.Errorf("no such ref: %v", req.Ref)
	}
	if ref == nil {
		return nil, errors.Wrapf(os.ErrNotExist, "%s not found", req.FilePath)
	}
	r, err := ref.Result(ctx)
	if err != nil {
		return nil, err
	}
	workerRef, ok := r.Sys().(*worker.WorkerRef)
	if !ok {
		return nil, errors.Errorf("invalid ref: %T", r.Sys())
	}

	newReq := cacheutil.ReadRequest{
		Filename: req.FilePath,
	}
	if r := req.Range; r != nil {
		newReq.Range = &cacheutil.FileRange{
			Offset: int(r.Offset),
			Length: int(r.Length),
		}
	}

	m, err := workerRef.ImmutableRef.Mount(ctx, true, session.NewGroup(lbf.sid))
	if err != nil {
		return nil, err
	}

	dt, err := cacheutil.ReadFile(ctx, m, newReq)
	if err != nil {
		return nil, err
	}

	return &pb.ReadFileResponse{Data: dt}, nil
}

func (lbf *llbBridgeForwarder) ReadDir(ctx context.Context, req *pb.ReadDirRequest) (*pb.ReadDirResponse, error) {
	ctx = tracing.ContextWithSpanFromContext(ctx, lbf.callCtx)
	lbf.mu.Lock()
	ref, ok := lbf.refs[req.Ref]
	lbf.mu.Unlock()
	if !ok {
		return nil, errors.Errorf("no such ref: %v", req.Ref)
	}
	if ref == nil {
		return nil, errors.Wrapf(os.ErrNotExist, "%s not found", req.DirPath)
	}
	r, err := ref.Result(ctx)
	if err != nil {
		return nil, err
	}
	workerRef, ok := r.Sys().(*worker.WorkerRef)
	if !ok {
		return nil, errors.Errorf("invalid ref: %T", r.Sys())
	}

	newReq := cacheutil.ReadDirRequest{
		Path:           req.DirPath,
		IncludePattern: req.IncludePattern,
	}
	m, err := workerRef.ImmutableRef.Mount(ctx, true, session.NewGroup(lbf.sid))
	if err != nil {
		return nil, err
	}
	entries, err := cacheutil.ReadDir(ctx, m, newReq)
	if err != nil {
		return nil, err
	}

	return &pb.ReadDirResponse{Entries: entries}, nil
}

func (lbf *llbBridgeForwarder) StatFile(ctx context.Context, req *pb.StatFileRequest) (*pb.StatFileResponse, error) {
	ctx = tracing.ContextWithSpanFromContext(ctx, lbf.callCtx)
	lbf.mu.Lock()
	ref, ok := lbf.refs[req.Ref]
	lbf.mu.Unlock()
	if !ok {
		return nil, errors.Errorf("no such ref: %v", req.Ref)
	}
	if ref == nil {
		return nil, errors.Wrapf(os.ErrNotExist, "%s not found", req.Path)
	}
	r, err := ref.Result(ctx)
	if err != nil {
		return nil, err
	}
	workerRef, ok := r.Sys().(*worker.WorkerRef)
	if !ok {
		return nil, errors.Errorf("invalid ref: %T", r.Sys())
	}
	m, err := workerRef.ImmutableRef.Mount(ctx, true, session.NewGroup(lbf.sid))
	if err != nil {
		return nil, err
	}
	st, err := cacheutil.StatFile(ctx, m, req.Path)
	if err != nil {
		return nil, err
	}

	return &pb.StatFileResponse{Stat: st}, nil
}

func (lbf *llbBridgeForwarder) Ping(context.Context, *pb.PingRequest) (*pb.PongResponse, error) {

	workers := lbf.workers.WorkerInfos()
	pbWorkers := make([]*apitypes.WorkerRecord, 0, len(workers))
	for _, w := range workers {
		pbWorkers = append(pbWorkers, &apitypes.WorkerRecord{
			ID:        w.ID,
			Labels:    w.Labels,
			Platforms: opspb.PlatformsFromSpec(w.Platforms),
		})
	}

	return &pb.PongResponse{
		FrontendAPICaps: pb.Caps.All(),
		Workers:         pbWorkers,
		LLBCaps:         opspb.Caps.All(),
	}, nil
}

func (lbf *llbBridgeForwarder) Return(ctx context.Context, in *pb.ReturnRequest) (*pb.ReturnResponse, error) {
	if in.Error != nil {
		return lbf.setResult(nil, grpcerrors.FromGRPC(status.ErrorProto(&spb.Status{
			Code:    in.Error.Code,
			Message: in.Error.Message,
			Details: convertGogoAny(in.Error.Details),
		})))
	}
	r := &frontend.Result{
		Metadata: in.Result.Metadata,
	}

	switch res := in.Result.Result.(type) {
	case *pb.Result_RefDeprecated:
		ref, err := lbf.convertRef(res.RefDeprecated)
		if err != nil {
			return nil, err
		}
		r.Ref = ref
	case *pb.Result_RefsDeprecated:
		m := map[string]solver.ResultProxy{}
		for k, id := range res.RefsDeprecated.Refs {
			ref, err := lbf.convertRef(id)
			if err != nil {
				return nil, err
			}
			m[k] = ref
		}
		r.Refs = m
	case *pb.Result_Ref:
		ref, err := lbf.convertRef(res.Ref.Id)
		if err != nil {
			return nil, err
		}
		r.Ref = ref
	case *pb.Result_Refs:
		m := map[string]solver.ResultProxy{}
		for k, ref := range res.Refs.Refs {
			ref, err := lbf.convertRef(ref.Id)
			if err != nil {
				return nil, err
			}
			m[k] = ref
		}
		r.Refs = m
	}
	return lbf.setResult(r, nil)
}

func (lbf *llbBridgeForwarder) Inputs(ctx context.Context, in *pb.InputsRequest) (*pb.InputsResponse, error) {
	return &pb.InputsResponse{
		Definitions: lbf.inputs,
	}, nil
}

func (lbf *llbBridgeForwarder) NewContainer(ctx context.Context, in *pb.NewContainerRequest) (_ *pb.NewContainerResponse, err error) {
	logrus.Debugf("|<--- NewContainer %s", in.ContainerID)
	ctrReq := NewContainerRequest{
		ContainerID: in.ContainerID,
		NetMode:     in.Network,
		Platform:    in.Platform,
		Constraints: in.Constraints,
	}

	for _, m := range in.Mounts {
		var refProxy solver.ResultProxy
		if m.ResultID != "" {
			refProxy, err = lbf.convertRef(m.ResultID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find ref %s for %q mount", m.ResultID, m.Dest)
			}
		}
		ctrReq.Mounts = append(ctrReq.Mounts, Mount{
			Dest:      m.Dest,
			Selector:  m.Selector,
			Readonly:  m.Readonly,
			MountType: m.MountType,
			RefProxy:  refProxy,
			CacheOpt:  m.CacheOpt,
			SecretOpt: m.SecretOpt,
			SSHOpt:    m.SSHOpt,
		})
	}

	// Not using `ctx` here because it will get cancelled as soon as NewContainer returns
	// and we want the context to live for the duration of the container.
	group := session.NewGroup(lbf.sid)
	ctr, err := NewContainer(context.Background(), lbf.llbBridge, lbf.sm, group, ctrReq)
	if err != nil {
		return nil, stack.Enable(err)
	}
	defer func() {
		if err != nil {
			ctr.Release(ctx) // ensure release on error
		}
	}()

	lbf.ctrsMu.Lock()
	defer lbf.ctrsMu.Unlock()
	// ensure we are not clobbering a dup container id request
	if _, ok := lbf.ctrs[in.ContainerID]; ok {
		return nil, stack.Enable(status.Errorf(codes.AlreadyExists, "Container %s already exists", in.ContainerID))
	}
	lbf.ctrs[in.ContainerID] = ctr
	return &pb.NewContainerResponse{}, nil
}

func (lbf *llbBridgeForwarder) ReleaseContainer(ctx context.Context, in *pb.ReleaseContainerRequest) (*pb.ReleaseContainerResponse, error) {
	logrus.Debugf("|<--- ReleaseContainer %s", in.ContainerID)
	lbf.ctrsMu.Lock()
	ctr, ok := lbf.ctrs[in.ContainerID]
	delete(lbf.ctrs, in.ContainerID)
	lbf.ctrsMu.Unlock()
	if !ok {
		return nil, errors.Errorf("container details for %s not found", in.ContainerID)
	}
	err := ctr.Release(ctx)
	return &pb.ReleaseContainerResponse{}, stack.Enable(err)
}

type processIO struct {
	id       string
	mu       sync.Mutex
	resize   func(context.Context, gwclient.WinSize) error
	done     chan struct{}
	doneOnce sync.Once
	// these track the process side of the io pipe for
	// read (fd=0) and write (fd=1, fd=2)
	processReaders map[uint32]io.ReadCloser
	processWriters map[uint32]io.WriteCloser
	// these track the server side of the io pipe, so
	// when we receive an EOF over grpc, we will close
	// this end
	serverWriters map[uint32]io.WriteCloser
	serverReaders map[uint32]io.ReadCloser
}

func newProcessIO(id string, openFds []uint32) *processIO {
	pio := &processIO{
		id:             id,
		processReaders: map[uint32]io.ReadCloser{},
		processWriters: map[uint32]io.WriteCloser{},
		serverReaders:  map[uint32]io.ReadCloser{},
		serverWriters:  map[uint32]io.WriteCloser{},
		done:           make(chan struct{}),
	}

	for _, fd := range openFds {
		// TODO do we know which way to pipe each fd?  For now assume fd0 is for
		// reading, and the rest are for writing
		r, w := io.Pipe()
		if fd == 0 {
			pio.processReaders[fd] = r
			pio.serverWriters[fd] = w
		} else {
			pio.processWriters[fd] = w
			pio.serverReaders[fd] = r
		}
	}

	return pio
}

func (pio *processIO) Close() (err error) {
	pio.mu.Lock()
	defer pio.mu.Unlock()
	for fd, r := range pio.processReaders {
		delete(pio.processReaders, fd)
		err1 := r.Close()
		if err1 != nil && err == nil {
			err = stack.Enable(err1)
		}
	}
	for fd, w := range pio.serverReaders {
		delete(pio.serverReaders, fd)
		err1 := w.Close()
		if err1 != nil && err == nil {
			err = stack.Enable(err1)
		}
	}
	pio.Done()
	return err
}

func (pio *processIO) Done() {
	stillOpen := len(pio.processReaders) + len(pio.processWriters) + len(pio.serverReaders) + len(pio.serverWriters)
	if stillOpen == 0 {
		pio.doneOnce.Do(func() {
			close(pio.done)
		})
	}
}

func (pio *processIO) Write(f *pb.FdMessage) (err error) {
	pio.mu.Lock()
	writer := pio.serverWriters[f.Fd]
	pio.mu.Unlock()
	if writer == nil {
		return status.Errorf(codes.OutOfRange, "fd %d unavailable to write", f.Fd)
	}
	defer func() {
		if err != nil || f.EOF {
			writer.Close()
			pio.mu.Lock()
			defer pio.mu.Unlock()
			delete(pio.serverWriters, f.Fd)
			pio.Done()
		}
	}()
	if len(f.Data) > 0 {
		_, err = writer.Write(f.Data)
		return stack.Enable(err)
	}
	return nil
}

type outputWriter struct {
	stream    pb.LLBBridge_ExecProcessServer
	fd        uint32
	processID string
}

func (w *outputWriter) Write(msg []byte) (int, error) {
	logrus.Debugf("|---> File Message %s, fd=%d, %d bytes", w.processID, w.fd, len(msg))
	err := w.stream.Send(&pb.ExecMessage{
		ProcessID: w.processID,
		Input: &pb.ExecMessage_File{
			File: &pb.FdMessage{
				Fd:   w.fd,
				Data: msg,
			},
		},
	})
	return len(msg), stack.Enable(err)
}

func (lbf *llbBridgeForwarder) ExecProcess(srv pb.LLBBridge_ExecProcessServer) error {
	eg, ctx := errgroup.WithContext(srv.Context())

	msgs := make(chan *pb.ExecMessage)

	eg.Go(func() error {
		defer close(msgs)
		for {
			execMsg, err := srv.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return stack.Enable(err)
			}
			switch m := execMsg.GetInput().(type) {
			case *pb.ExecMessage_Init:
				logrus.Debugf("|<--- Init Message %s", execMsg.ProcessID)
			case *pb.ExecMessage_File:
				if m.File.EOF {
					logrus.Debugf("|<--- File Message %s, fd=%d, EOF", execMsg.ProcessID, m.File.Fd)
				} else {
					logrus.Debugf("|<--- File Message %s, fd=%d, %d bytes", execMsg.ProcessID, m.File.Fd, len(m.File.Data))
				}
			case *pb.ExecMessage_Resize:
				logrus.Debugf("|<--- Resize Message %s", execMsg.ProcessID)
			}
			select {
			case <-ctx.Done():
			case msgs <- execMsg:
			}
		}
	})

	eg.Go(func() error {
		pios := make(map[string]*processIO)
		// close any stray pios on exit to make sure
		// all the associated resources get cleaned up
		defer func() {
			for _, pio := range pios {
				pio.Close()
			}
		}()

		for {
			var execMsg *pb.ExecMessage
			select {
			case <-ctx.Done():
				return nil
			case execMsg = <-msgs:
			}
			if execMsg == nil {
				return nil
			}

			pid := execMsg.ProcessID
			if pid == "" {
				return stack.Enable(status.Errorf(codes.InvalidArgument, "ProcessID required"))
			}

			pio, pioFound := pios[pid]

			if data := execMsg.GetFile(); data != nil {
				if !pioFound {
					return stack.Enable(status.Errorf(codes.NotFound, "IO for process %q not found", pid))
				}
				err := pio.Write(data)
				if err != nil {
					return stack.Enable(err)
				}
			} else if resize := execMsg.GetResize(); resize != nil {
				if !pioFound {
					return stack.Enable(status.Errorf(codes.NotFound, "IO for process %q not found", pid))
				}
				pio.resize(ctx, gwclient.WinSize{
					Cols: resize.Cols,
					Rows: resize.Rows,
				})
			} else if init := execMsg.GetInit(); init != nil {
				if pioFound {
					return stack.Enable(status.Errorf(codes.AlreadyExists, "Process %s already exists", pid))
				}
				id := init.ContainerID
				lbf.ctrsMu.Lock()
				ctr, ok := lbf.ctrs[id]
				lbf.ctrsMu.Unlock()
				if !ok {
					return stack.Enable(status.Errorf(codes.NotFound, "container %q previously released or not created", id))
				}

				initCtx, initCancel := context.WithCancel(context.Background())
				defer initCancel()

				pio := newProcessIO(pid, init.Fds)
				pios[pid] = pio

				proc, err := ctr.Start(initCtx, gwclient.StartRequest{
					Args:   init.Meta.Args,
					Env:    init.Meta.Env,
					User:   init.Meta.User,
					Cwd:    init.Meta.Cwd,
					Tty:    init.Tty,
					Stdin:  pio.processReaders[0],
					Stdout: pio.processWriters[1],
					Stderr: pio.processWriters[2],
				})
				if err != nil {
					return stack.Enable(err)
				}
				pio.resize = proc.Resize

				eg.Go(func() error {
					<-pio.done
					logrus.Debugf("|---> Done Message %s", pid)
					err := srv.Send(&pb.ExecMessage{
						ProcessID: pid,
						Input: &pb.ExecMessage_Done{
							Done: &pb.DoneMessage{},
						},
					})
					return stack.Enable(err)
				})

				eg.Go(func() error {
					defer func() {
						pio.Close()
					}()
					err := proc.Wait()

					var statusCode uint32
					var exitError *errdefs.ExitError
					var statusError *rpc.Status
					if err != nil {
						statusCode = containerd.UnknownExitStatus
						st, _ := status.FromError(grpcerrors.ToGRPC(err))
						stp := st.Proto()
						statusError = &rpc.Status{
							Code:    stp.Code,
							Message: stp.Message,
							Details: convertToGogoAny(stp.Details),
						}
					}
					if errors.As(err, &exitError) {
						statusCode = exitError.ExitCode
					}
					logrus.Debugf("|---> Exit Message %s, code=%d, error=%s", pid, statusCode, err)
					sendErr := srv.Send(&pb.ExecMessage{
						ProcessID: pid,
						Input: &pb.ExecMessage_Exit{
							Exit: &pb.ExitMessage{
								Code:  statusCode,
								Error: statusError,
							},
						},
					})

					if sendErr != nil && err != nil {
						return errors.Wrap(sendErr, err.Error())
					} else if sendErr != nil {
						return stack.Enable(sendErr)
					}

					if err != nil && statusCode != 0 {
						// this was a container exit error which is "normal" so
						// don't return this error from the errgroup
						return nil
					}
					return stack.Enable(err)
				})

				logrus.Debugf("|---> Started Message %s", pid)
				err = srv.Send(&pb.ExecMessage{
					ProcessID: pid,
					Input: &pb.ExecMessage_Started{
						Started: &pb.StartedMessage{},
					},
				})
				if err != nil {
					return stack.Enable(err)
				}

				// start sending Fd output back to client, this is done after
				// StartedMessage so that Fd output will not potentially arrive
				// to the client before "Started" as the container starts up.
				for fd, file := range pio.serverReaders {
					fd, file := fd, file
					eg.Go(func() error {
						defer func() {
							file.Close()
							pio.mu.Lock()
							defer pio.mu.Unlock()
							w := pio.processWriters[fd]
							if w != nil {
								w.Close()
							}
							delete(pio.processWriters, fd)
							pio.Done()
						}()
						dest := &outputWriter{
							stream:    srv,
							fd:        uint32(fd),
							processID: pid,
						}
						_, err := io.Copy(dest, file)
						// ignore ErrClosedPipe, it is EOF for our usage.
						if err != nil && !errors.Is(err, io.ErrClosedPipe) {
							return stack.Enable(err)
						}
						// no error so must be EOF
						logrus.Debugf("|---> File Message %s, fd=%d, EOF", pid, fd)
						err = srv.Send(&pb.ExecMessage{
							ProcessID: pid,
							Input: &pb.ExecMessage_File{
								File: &pb.FdMessage{
									Fd:  uint32(fd),
									EOF: true,
								},
							},
						})
						return stack.Enable(err)
					})
				}
			}
		}
	})

	err := eg.Wait()
	return stack.Enable(err)
}

func (lbf *llbBridgeForwarder) convertRef(id string) (solver.ResultProxy, error) {
	if id == "" {
		return nil, nil
	}

	lbf.mu.Lock()
	defer lbf.mu.Unlock()

	r, ok := lbf.refs[id]
	if !ok {
		return nil, errors.Errorf("return reference %s not found", id)
	}

	return r, nil
}

func serve(ctx context.Context, grpcServer *grpc.Server, conn net.Conn) {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	logrus.Debugf("serving grpc connection")
	(&http2.Server{}).ServeConn(conn, &http2.ServeConnOpts{Handler: grpcServer})
}

type markTypeFrontend struct{}

func (*markTypeFrontend) SetImageOption(ii *llb.ImageInfo) {
	ii.RecordType = string(client.UsageRecordTypeFrontend)
}

func convertGogoAny(in []*gogotypes.Any) []*any.Any {
	out := make([]*any.Any, len(in))
	for i := range in {
		out[i] = &any.Any{TypeUrl: in[i].TypeUrl, Value: in[i].Value}
	}
	return out
}

func convertToGogoAny(in []*any.Any) []*gogotypes.Any {
	out := make([]*gogotypes.Any, len(in))
	for i := range in {
		out[i] = &gogotypes.Any{TypeUrl: in[i].TypeUrl, Value: in[i].Value}
	}
	return out
}

func getCaps(label string) map[string]struct{} {
	if label == "" {
		return make(map[string]struct{})
	}
	caps := strings.Split(label, ",")
	out := make(map[string]struct{}, len(caps))
	for _, c := range caps {
		name := strings.SplitN(c, "+", 2)
		if name[0] != "" {
			out[name[0]] = struct{}{}
		}
	}
	return out
}

func addCapsForKnownFrontends(caps map[string]struct{}, dgst digest.Digest) {
	// these frontends were built without caps detection but do support inputs
	defaults := map[digest.Digest]struct{}{
		"sha256:9ac1c43a60e31dca741a6fe8314130a9cd4c4db0311fbbc636ff992ef60ae76d": {}, // docker/dockerfile:1.1.6
		"sha256:080bd74d8778f83e7b670de193362d8c593c8b14f5c8fb919d28ee8feda0d069": {}, // docker/dockerfile:1.1.7
		"sha256:60543a9d92b92af5088fb2938fb09b2072684af8384399e153e137fe081f8ab4": {}, // docker/dockerfile:1.1.6-experimental
		"sha256:de85b2f3a3e8a2f7fe48e8e84a65f6fdd5cd5183afa6412fff9caa6871649c44": {}, // docker/dockerfile:1.1.7-experimental
	}
	if _, ok := defaults[dgst]; ok {
		caps["moby.buildkit.frontend.inputs"] = struct{}{}
	}
}
