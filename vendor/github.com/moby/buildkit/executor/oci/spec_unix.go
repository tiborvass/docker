// +build !windows

package oci

import (
	"github.com/containerd/containerd/contrib/seccomp"
	"github.com/containerd/containerd/oci"
	"github.com/docker/docker/pkg/idtools"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/entitlements/security"
	"github.com/moby/buildkit/util/system"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func generateMountOpts(resolvConf, hostsFile string) ([]oci.SpecOpts, error) {
	return []oci.SpecOpts{
		// https://github.com/moby/buildkit/issues/429
		withRemovedMount("/run"),
		withROBind(resolvConf, "/etc/resolv.conf"),
		withROBind(hostsFile, "/etc/hosts"),
		withCGroup(),
	}, nil
}

// generateSecurityOpts may affect mounts, so must be called after generateMountOpts
func generateSecurityOpts(mode pb.SecurityMode) ([]oci.SpecOpts, error) {
	if mode == pb.SecurityMode_INSECURE {
		return []oci.SpecOpts{
			security.WithInsecureSpec(),
			oci.WithWriteableCgroupfs,
			oci.WithWriteableSysfs,
		}, nil
	} else if system.SeccompSupported() && mode == pb.SecurityMode_SANDBOX {
		return []oci.SpecOpts{seccomp.WithDefaultProfile()}, nil
	}
	return nil, nil
}

// generateProcessModeOpts may affect mounts, so must be called after generateMountOpts
func generateProcessModeOpts(mode ProcessMode) ([]oci.SpecOpts, error) {
	if mode == NoProcessSandbox {
		return []oci.SpecOpts{
			oci.WithHostNamespace(specs.PIDNamespace),
			withBoundProc(),
		}, nil
		// TODO(AkihiroSuda): Configure seccomp to disable ptrace (and prctl?) explicitly
	}
	return nil, nil
}

func generateIDmapOpts(idmap *idtools.IdentityMapping) ([]oci.SpecOpts, error) {
	if idmap == nil {
		return nil, nil
	}
	return []oci.SpecOpts{
		oci.WithUserNamespace(specMapping(idmap.UIDs()), specMapping(idmap.GIDs())),
	}, nil
}
