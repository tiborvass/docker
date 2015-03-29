package execdriver

import (
	"fmt"
	"strings"

	"github.com/tiborvass/docker/pkg/stringutils"
	"github.com/syndtr/gocapability/capability"
)

var capabilityList = Capabilities{
	{Key: "SETPCAP", Value: capability.CAP_SETPCAP},
	{Key: "SYS_MODULE", Value: capability.CAP_SYS_MODULE},
	{Key: "SYS_RAWIO", Value: capability.CAP_SYS_RAWIO},
	{Key: "SYS_PACCT", Value: capability.CAP_SYS_PACCT},
	{Key: "SYS_ADMIN", Value: capability.CAP_SYS_ADMIN},
	{Key: "SYS_NICE", Value: capability.CAP_SYS_NICE},
	{Key: "SYS_RESOURCE", Value: capability.CAP_SYS_RESOURCE},
	{Key: "SYS_TIME", Value: capability.CAP_SYS_TIME},
	{Key: "SYS_TTY_CONFIG", Value: capability.CAP_SYS_TTY_CONFIG},
	{Key: "MKNOD", Value: capability.CAP_MKNOD},
	{Key: "AUDIT_WRITE", Value: capability.CAP_AUDIT_WRITE},
	{Key: "AUDIT_CONTROL", Value: capability.CAP_AUDIT_CONTROL},
	{Key: "MAC_OVERRIDE", Value: capability.CAP_MAC_OVERRIDE},
	{Key: "MAC_ADMIN", Value: capability.CAP_MAC_ADMIN},
	{Key: "NET_ADMIN", Value: capability.CAP_NET_ADMIN},
	{Key: "SYSLOG", Value: capability.CAP_SYSLOG},
	{Key: "CHOWN", Value: capability.CAP_CHOWN},
	{Key: "NET_RAW", Value: capability.CAP_NET_RAW},
	{Key: "DAC_OVERRIDE", Value: capability.CAP_DAC_OVERRIDE},
	{Key: "FOWNER", Value: capability.CAP_FOWNER},
	{Key: "DAC_READ_SEARCH", Value: capability.CAP_DAC_READ_SEARCH},
	{Key: "FSETID", Value: capability.CAP_FSETID},
	{Key: "KILL", Value: capability.CAP_KILL},
	{Key: "SETGID", Value: capability.CAP_SETGID},
	{Key: "SETUID", Value: capability.CAP_SETUID},
	{Key: "LINUX_IMMUTABLE", Value: capability.CAP_LINUX_IMMUTABLE},
	{Key: "NET_BIND_SERVICE", Value: capability.CAP_NET_BIND_SERVICE},
	{Key: "NET_BROADCAST", Value: capability.CAP_NET_BROADCAST},
	{Key: "IPC_LOCK", Value: capability.CAP_IPC_LOCK},
	{Key: "IPC_OWNER", Value: capability.CAP_IPC_OWNER},
	{Key: "SYS_CHROOT", Value: capability.CAP_SYS_CHROOT},
	{Key: "SYS_PTRACE", Value: capability.CAP_SYS_PTRACE},
	{Key: "SYS_BOOT", Value: capability.CAP_SYS_BOOT},
	{Key: "LEASE", Value: capability.CAP_LEASE},
	{Key: "SETFCAP", Value: capability.CAP_SETFCAP},
	{Key: "WAKE_ALARM", Value: capability.CAP_WAKE_ALARM},
	{Key: "BLOCK_SUSPEND", Value: capability.CAP_BLOCK_SUSPEND},
}

type (
	CapabilityMapping struct {
		Key   string         `json:"key,omitempty"`
		Value capability.Cap `json:"value,omitempty"`
	}
	Capabilities []*CapabilityMapping
)

func (c *CapabilityMapping) String() string {
	return c.Key
}

func GetCapability(key string) *CapabilityMapping {
	for _, capp := range capabilityList {
		if capp.Key == key {
			cpy := *capp
			return &cpy
		}
	}
	return nil
}

func GetAllCapabilities() []string {
	output := make([]string, len(capabilityList))
	for i, capability := range capabilityList {
		output[i] = capability.String()
	}
	return output
}

func TweakCapabilities(basics, adds, drops []string) ([]string, error) {
	var (
		newCaps []string
		allCaps = GetAllCapabilities()
	)

	// look for invalid cap in the drop list
	for _, cap := range drops {
		if strings.ToLower(cap) == "all" {
			continue
		}
		if !stringutils.InSlice(allCaps, cap) {
			return nil, fmt.Errorf("Unknown capability drop: %q", cap)
		}
	}

	// handle --cap-add=all
	if stringutils.InSlice(adds, "all") {
		basics = allCaps
	}

	if !stringutils.InSlice(drops, "all") {
		for _, cap := range basics {
			// skip `all` aready handled above
			if strings.ToLower(cap) == "all" {
				continue
			}

			// if we don't drop `all`, add back all the non-dropped caps
			if !stringutils.InSlice(drops, cap) {
				newCaps = append(newCaps, strings.ToUpper(cap))
			}
		}
	}

	for _, cap := range adds {
		// skip `all` aready handled above
		if strings.ToLower(cap) == "all" {
			continue
		}

		if !stringutils.InSlice(allCaps, cap) {
			return nil, fmt.Errorf("Unknown capability to add: %q", cap)
		}

		// add cap if not already in the list
		if !stringutils.InSlice(newCaps, cap) {
			newCaps = append(newCaps, strings.ToUpper(cap))
		}
	}

	return newCaps, nil
}
