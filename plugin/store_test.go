package plugin

import (
	"testing"

	"github.com/moby/moby-core/api/types"
	"github.com/moby/moby-core/plugin/v2"
)

func TestFilterByCapNeg(t *testing.T) {
	p := v2.Plugin{PluginObj: types.Plugin{Name: "test:latest"}}
	iType := types.PluginInterfaceType{Capability: "volumedriver", Prefix: "docker", Version: "1.0"}
	i := types.PluginConfigInterface{Socket: "plugins.sock", Types: []types.PluginInterfaceType{iType}}
	p.PluginObj.Config.Interface = i

	_, err := p.FilterByCap("foobar")
	if err == nil {
		t.Fatalf("expected inadequate error, got %v", err)
	}
}

func TestFilterByCapPos(t *testing.T) {
	p := v2.Plugin{PluginObj: types.Plugin{Name: "test:latest"}}

	iType := types.PluginInterfaceType{Capability: "volumedriver", Prefix: "docker", Version: "1.0"}
	i := types.PluginConfigInterface{Socket: "plugins.sock", Types: []types.PluginInterfaceType{iType}}
	p.PluginObj.Config.Interface = i

	_, err := p.FilterByCap("volumedriver")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
