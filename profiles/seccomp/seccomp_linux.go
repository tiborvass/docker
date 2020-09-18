//go:generate go run -tags 'seccomp' generate.go

package seccomp // import "github.com/tiborvass/docker/profiles/seccomp"

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/tiborvass/docker/pkg/parsers/kernel"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// GetDefaultProfile returns the default seccomp profile.
func GetDefaultProfile(rs *specs.Spec) (*specs.LinuxSeccomp, error) {
	return setupSeccomp(DefaultProfile(), rs)
}

// LoadProfile takes a json string and decodes the seccomp profile.
func LoadProfile(body string, rs *specs.Spec) (*specs.LinuxSeccomp, error) {
	var config Seccomp
	if err := json.Unmarshal([]byte(body), &config); err != nil {
		return nil, fmt.Errorf("Decoding seccomp profile failed: %v", err)
	}
	return setupSeccomp(&config, rs)
}

// libseccomp string => seccomp arch
var nativeToSeccomp = map[string]Arch{
	"x86":         ArchX86,
	"amd64":       ArchX86_64,
	"arm":         ArchARM,
	"arm64":       ArchAARCH64,
	"mips64":      ArchMIPS64,
	"mips64n32":   ArchMIPS64N32,
	"mipsel64":    ArchMIPSEL64,
	"mips3l64n32": ArchMIPSEL64N32,
	"mipsle":      ArchMIPSEL,
	"ppc":         ArchPPC,
	"ppc64":       ArchPPC64,
	"ppc64le":     ArchPPC64LE,
	"s390":        ArchS390,
	"s390x":       ArchS390X,
}

// GOARCH => libseccomp string
var goToNative = map[string]string{
	"386":         "x86",
	"amd64":       "amd64",
	"arm":         "arm",
	"arm64":       "arm64",
	"mips64":      "mips64",
	"mips64p32":   "mips64n32",
	"mips64le":    "mipsel64",
	"mips64p32le": "mips3l64n32",
	"mipsle":      "mipsel",
	"ppc":         "ppc",
	"ppc64":       "ppc64",
	"ppc64le":     "ppc64le",
	"s390":        "s390",
	"s390x":       "s390x",
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

func setupSeccomp(config *Seccomp, rs *specs.Spec) (*specs.LinuxSeccomp, error) {
	if config == nil {
		return nil, nil
	}

	// No default action specified, no syscalls listed, assume seccomp disabled
	if config.DefaultAction == "" && len(config.Syscalls) == 0 {
		return nil, nil
	}

	newConfig := &specs.LinuxSeccomp{}

	if len(config.Architectures) != 0 && len(config.ArchMap) != 0 {
		return nil, errors.New("'architectures' and 'archMap' were specified in the seccomp profile, use either 'architectures' or 'archMap'")
	}

	// if config.Architectures == 0 then libseccomp will figure out the architecture to use
	if len(config.Architectures) != 0 {
		for _, a := range config.Architectures {
			newConfig.Architectures = append(newConfig.Architectures, specs.Arch(a))
		}
	}

	arch := goToNative[runtime.GOARCH]
	seccompArch, archExists := nativeToSeccomp[arch]

	if len(config.ArchMap) != 0 && archExists {
		for _, a := range config.ArchMap {
			if a.Arch == seccompArch {
				newConfig.Architectures = append(newConfig.Architectures, specs.Arch(a.Arch))
				for _, sa := range a.SubArches {
					newConfig.Architectures = append(newConfig.Architectures, specs.Arch(sa))
				}
				break
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

func createSpecsSyscall(names []string, action Action, args []*Arg) specs.LinuxSyscall {
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
