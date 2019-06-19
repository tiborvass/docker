// +build linux

package seccomp // import "github.com/tiborvass/docker/profiles/seccomp"

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/pkg/parsers/kernel"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	libseccomp "github.com/seccomp/libseccomp-golang"
)

//go:generate go run -tags 'seccomp' generate.go

// GetDefaultProfile returns the default seccomp profile.
func GetDefaultProfile(rs *specs.Spec) (*specs.LinuxSeccomp, error) {
	return setupSeccomp(DefaultProfile(), rs)
}

// LoadProfile takes a json string and decodes the seccomp profile.
func LoadProfile(body string, rs *specs.Spec) (*specs.LinuxSeccomp, error) {
	var config types.Seccomp
	if err := json.Unmarshal([]byte(body), &config); err != nil {
		return nil, fmt.Errorf("Decoding seccomp profile failed: %v", err)
	}
	return setupSeccomp(&config, rs)
}

var nativeToSeccomp = map[string]types.Arch{
	"amd64":       types.ArchX86_64,
	"arm64":       types.ArchAARCH64,
	"mips64":      types.ArchMIPS64,
	"mips64n32":   types.ArchMIPS64N32,
	"mipsel64":    types.ArchMIPSEL64,
	"mipsel64n32": types.ArchMIPSEL64N32,
	"s390x":       types.ArchS390X,
}

// inSlice tests whether a string is contained in a slice of strings or not.
// Comparison is case sensitive
func inSlice(slice []string, s string) bool {
	for _, ss := range slice {
		if s == ss {
			return true
		}
	}
	return false
}

func setupSeccomp(config *types.Seccomp, rs *specs.Spec) (*specs.LinuxSeccomp, error) {
	if config == nil {
		return nil, nil
	}

	// No default action specified, no syscalls listed, assume seccomp disabled
	if config.DefaultAction == "" && len(config.Syscalls) == 0 {
		return nil, nil
	}

	newConfig := &specs.LinuxSeccomp{}

	var arch string
	var native, err = libseccomp.GetNativeArch()
	if err == nil {
		arch = native.String()
	}

	if len(config.Architectures) != 0 && len(config.ArchMap) != 0 {
		return nil, errors.New("'architectures' and 'archMap' were specified in the seccomp profile, use either 'architectures' or 'archMap'")
	}

	// if config.Architectures == 0 then libseccomp will figure out the architecture to use
	if len(config.Architectures) != 0 {
		for _, a := range config.Architectures {
			newConfig.Architectures = append(newConfig.Architectures, specs.Arch(a))
		}
	}

	if len(config.ArchMap) != 0 {
		for _, a := range config.ArchMap {
			seccompArch, ok := nativeToSeccomp[arch]
			if ok {
				if a.Arch == seccompArch {
					newConfig.Architectures = append(newConfig.Architectures, specs.Arch(a.Arch))
					for _, sa := range a.SubArches {
						newConfig.Architectures = append(newConfig.Architectures, specs.Arch(sa))
					}
					break
				}
			}
		}
	}

	newConfig.DefaultAction = specs.LinuxSeccompAction(config.DefaultAction)

Loop:
	// Loop through all syscall blocks and convert them to libcontainer format after filtering them
	for _, call := range config.Syscalls {
		if len(call.Excludes.Arches) > 0 {
			if inSlice(call.Excludes.Arches, arch) {
				continue Loop
			}
		}
		if len(call.Excludes.Caps) > 0 {
			for _, c := range call.Excludes.Caps {
				if inSlice(rs.Process.Capabilities.Bounding, c) {
					continue Loop
				}
			}
		}
		if call.Excludes.MinKernel != "" {
			if ok, err := kernelGreaterEqualThan(call.Excludes.MinKernel); err != nil {
				return nil, err
			} else if ok {
				continue Loop
			}
		}
		if len(call.Includes.Arches) > 0 {
			if !inSlice(call.Includes.Arches, arch) {
				continue Loop
			}
		}
		if len(call.Includes.Caps) > 0 {
			for _, c := range call.Includes.Caps {
				if !inSlice(rs.Process.Capabilities.Bounding, c) {
					continue Loop
				}
			}
		}
		if call.Includes.MinKernel != "" {
			if ok, err := kernelGreaterEqualThan(call.Includes.MinKernel); err != nil {
				return nil, err
			} else if !ok {
				continue Loop
			}
		}

		if call.Name != "" && len(call.Names) != 0 {
			return nil, errors.New("'name' and 'names' were specified in the seccomp profile, use either 'name' or 'names'")
		}

		if call.Name != "" {
			newConfig.Syscalls = append(newConfig.Syscalls, createSpecsSyscall([]string{call.Name}, call.Action, call.Args))
		} else {
			newConfig.Syscalls = append(newConfig.Syscalls, createSpecsSyscall(call.Names, call.Action, call.Args))
		}
	}

	return newConfig, nil
}

func createSpecsSyscall(names []string, action types.Action, args []*types.Arg) specs.LinuxSyscall {
	newCall := specs.LinuxSyscall{
		Names:  names,
		Action: specs.LinuxSeccompAction(action),
	}

	// Loop through all the arguments of the syscall and convert them
	for _, arg := range args {
		newArg := specs.LinuxSeccompArg{
			Index:    arg.Index,
			Value:    arg.Value,
			ValueTwo: arg.ValueTwo,
			Op:       specs.LinuxSeccompOperator(arg.Op),
		}

		newCall.Args = append(newCall.Args, newArg)
	}
	return newCall
}

var currentKernelVersion *kernel.VersionInfo

func kernelGreaterEqualThan(v string) (bool, error) {
	version, err := kernel.ParseRelease(v)
	if err != nil {
		return false, err
	}
	if currentKernelVersion == nil {
		currentKernelVersion, err = kernel.GetKernelVersion()
		if err != nil {
			return false, err
		}
	}
	return kernel.CompareKernelVersion(*version, *currentKernelVersion) <= 0, nil
}
