package plugin_test

import (
	"net/http"
	"testing"

	"pogonian/internal/plugin"
)

type samplePlugin struct {
	plugin.BasePlugin
	initCalled   bool
	startCalled  bool
	reqCalled    bool
	closedCalled bool
}

func (s *samplePlugin) OnInit(ctx *plugin.PluginContext) error {
	s.initCalled = true
	return nil
}

func (s *samplePlugin) OnServerStart(ctx *plugin.PluginContext) error {
	s.startCalled = true
	return nil
}

func (s *samplePlugin) OnRequest(r *http.Request) error {
	s.reqCalled = true
	return nil
}

func (s *samplePlugin) OnShutdown() error {
	s.closedCalled = true
	return nil
}

func TestPlugin_Lifecycle(t *testing.T) {
	reg := plugin.NewRegistry()
	p := &samplePlugin{
		BasePlugin: plugin.BasePlugin{
			PluginName:    "analytics",
			PluginVersion: "1.0.0",
		},
	}

	// 1. Register & Init
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !p.initCalled {
		t.Errorf("Expected OnInit to be called on registration")
	}

	// 2. Server Start Hook
	if err := reg.TriggerServerStart(); err != nil {
		t.Fatalf("TriggerServerStart failed: %v", err)
	}
	if !p.startCalled {
		t.Errorf("Expected OnServerStart to be called")
	}

	// 3. Request Hook
	req, _ := http.NewRequest("GET", "/products", nil)
	if err := reg.TriggerRequest(req); err != nil {
		t.Fatalf("TriggerRequest failed: %v", err)
	}
	if !p.reqCalled {
		t.Errorf("Expected OnRequest to be called")
	}

	// 4. Shutdown Hook
	reg.TriggerShutdown()
	if !p.closedCalled {
		t.Errorf("Expected OnShutdown to be called")
	}
}
