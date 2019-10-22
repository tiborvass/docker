package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tiborvass/docker/api/types/filters"
)

// BuilderGCRule represents a GC rule for buildkit cache
type BuilderGCRule struct {
	All         bool            `json:",omitempty"`
	Filter      BuilderGCFilter `json:",omitempty"`
	KeepStorage string          `json:",omitempty"`
}

type BuilderGCFilter filters.Args

func (x *BuilderGCFilter) MarshalJSON() ([]byte, error) {
	f := filters.Args(*x)
	keys := f.Keys()
	sort.Strings(keys)
	arr := make([]string, 0, len(keys))
	for _, k := range keys {
		values := f.Get(k)
		for _, v := range values {
			arr = append(arr, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return json.Marshal(arr)
}

func (x *BuilderGCFilter) UnmarshalJSON(data []byte) error {
	var arr []string
	f := filters.NewArgs()
	if err := json.Unmarshal(data, &arr); err != nil {
		// backwards compat for deprecated buggy form
		err := json.Unmarshal(data, &f)
		*x = BuilderGCFilter(f)
		return err
	}
	for _, s := range arr {
		fields := strings.SplitN(s, "=", 2)
		name := strings.ToLower(strings.TrimSpace(fields[0]))
		value := strings.TrimSpace(fields[1])
		f.Add(name, value)
	}
	*x = BuilderGCFilter(f)
	return nil
}

// BuilderGCConfig contains GC config for a buildkit builder
type BuilderGCConfig struct {
	Enabled            bool            `json:",omitempty"`
	Policy             []BuilderGCRule `json:",omitempty"`
	DefaultKeepStorage string          `json:",omitempty"`
}

// BuilderConfig contains config for the builder
type BuilderConfig struct {
	GC BuilderGCConfig `json:",omitempty"`
}
