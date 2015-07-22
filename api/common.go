package api

import (
	"fmt"
	"mime"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/pkg/system"
	"github.com/tiborvass/docker/pkg/version"
	"github.com/docker/libtrust"
)

// Common constants for daemon and client.
const (
	// Version of Current REST API
	Version version.Version = "1.21"

	// MinVersion represents Minimun REST API version supported
	MinVersion version.Version = "1.12"

	// DefaultDockerfileName is the Default filename with Docker commands, read by docker build
	DefaultDockerfileName string = "Dockerfile"
)

// byPrivatePort is temporary type used to sort types.Port by PrivatePort
type byPrivatePort []types.Port

func (r byPrivatePort) Len() int           { return len(r) }
func (r byPrivatePort) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r byPrivatePort) Less(i, j int) bool { return r[i].PrivatePort < r[j].PrivatePort }

// DisplayablePorts returns formatted string representing open ports of container
// e.g. "0.0.0.0:80->9090/tcp, 9988/tcp"
// it's used by command 'docker ps'
func DisplayablePorts(ports []types.Port) string {
	var (
		result          = []string{}
		hostMappings    = []string{}
		firstInGroupMap map[string]int
		lastInGroupMap  map[string]int
	)
	firstInGroupMap = make(map[string]int)
	lastInGroupMap = make(map[string]int)
	sort.Sort(byPrivatePort(ports))
	for _, port := range ports {
		var (
			current      = port.PrivatePort
			portKey      = port.Type
			firstInGroup int
			lastInGroup  int
		)
		if port.IP != "" {
			if port.PublicPort != current {
				hostMappings = append(hostMappings, fmt.Sprintf("%s:%d->%d/%s", port.IP, port.PublicPort, port.PrivatePort, port.Type))
				continue
			}
			portKey = fmt.Sprintf("%s/%s", port.IP, port.Type)
		}
		firstInGroup = firstInGroupMap[portKey]
		lastInGroup = lastInGroupMap[portKey]

		if firstInGroup == 0 {
			firstInGroupMap[portKey] = current
			lastInGroupMap[portKey] = current
			continue
		}

		if current == (lastInGroup + 1) {
			lastInGroupMap[portKey] = current
			continue
		}
		result = append(result, formGroup(portKey, firstInGroup, lastInGroup))
		firstInGroupMap[portKey] = current
		lastInGroupMap[portKey] = current
	}
	for portKey, firstInGroup := range firstInGroupMap {
		result = append(result, formGroup(portKey, firstInGroup, lastInGroupMap[portKey]))
	}
	result = append(result, hostMappings...)
	return strings.Join(result, ", ")
}

func formGroup(key string, start, last int) string {
	var (
		group     string
		parts     = strings.Split(key, "/")
		groupType = parts[0]
		ip        = ""
	)
	if len(parts) > 1 {
		ip = parts[0]
		groupType = parts[1]
	}
	if start == last {
		group = fmt.Sprintf("%d", start)
	} else {
		group = fmt.Sprintf("%d-%d", start, last)
	}
	if ip != "" {
		group = fmt.Sprintf("%s:%s->%s", ip, group, group)
	}
	return fmt.Sprintf("%s/%s", group, groupType)
}

// MatchesContentType validates the content type against the expected one
func MatchesContentType(contentType, expectedType string) bool {
	mimetype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		logrus.Errorf("Error parsing media type: %s error: %v", contentType, err)
	}
	return err == nil && mimetype == expectedType
}

// LoadOrCreateTrustKey attempts to load the libtrust key at the given path,
// otherwise generates a new one
func LoadOrCreateTrustKey(trustKeyPath string) (libtrust.PrivateKey, error) {
	err := system.MkdirAll(filepath.Dir(trustKeyPath), 0700)
	if err != nil {
		return nil, err
	}
	trustKey, err := libtrust.LoadKeyFile(trustKeyPath)
	if err == libtrust.ErrKeyFileDoesNotExist {
		trustKey, err = libtrust.GenerateECP256PrivateKey()
		if err != nil {
			return nil, fmt.Errorf("Error generating key: %s", err)
		}
		if err := libtrust.SaveKey(trustKeyPath, trustKey); err != nil {
			return nil, fmt.Errorf("Error saving key file: %s", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("Error loading key file %s: %s", trustKeyPath, err)
	}
	return trustKey, nil
}
