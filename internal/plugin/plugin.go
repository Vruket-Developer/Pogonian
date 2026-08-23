package plugin

import (
	"fmt"
	"net/http"
	"sync"
)

// PluginContext provides plugins with project metadata and configuration hooks.
type PluginContext struct {
	ProjectDir  string
	Environment string
	Data        map[string]any
}

// Plugin defines the lifecycle contract for Pogonian extensions.
type Plugin interface {
	Name() string
	Version() string
	OnInit(ctx *PluginContext) error
	OnServerStart(ctx *PluginContext) error
	OnRequest(r *http.Request) error
	OnShutdown() error
}

// BasePlugin provides default no-op implementations of Plugin methods.
type BasePlugin struct {
	PluginName    string
	PluginVersion string
}

func (b *BasePlugin) Name() string                     { return b.PluginName }
func (b *BasePlugin) Version() string                  { return b.PluginVersion }
func (b *BasePlugin) OnInit(ctx *PluginContext) error  { return nil }
func (b *BasePlugin) OnServerStart(ctx *PluginContext) error { return nil }
func (b *BasePlugin) OnRequest(r *http.Request) error  { return nil }
func (b *BasePlugin) OnShutdown() error                { return nil }

// Registry manages the registration and execution of plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins []Plugin
	ctx     *PluginContext
}

// GlobalRegistry is the default framework plugin registry.
var GlobalRegistry = NewRegistry()

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make([]Plugin, 0),
		ctx: &PluginContext{
			Data: make(map[string]any),
		},
	}
}

// Register registers a plugin into the registry.
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.plugins {
		if existing.Name() == p.Name() {
			return fmt.Errorf("plugin '%s' is already registered", p.Name())
		}
	}

	r.plugins = append(r.plugins, p)
	return p.OnInit(r.ctx)
}

// TriggerServerStart notifies all plugins that the HTTP server is starting.
func (r *Registry) TriggerServerStart() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if err := p.OnServerStart(r.ctx); err != nil {
			return fmt.Errorf("plugin '%s' server start failed: %w", p.Name(), err)
		}
	}
	return nil
}

// TriggerRequest notifies all plugins of an incoming HTTP request.
func (r *Registry) TriggerRequest(req *http.Request) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if err := p.OnRequest(req); err != nil {
			return err
		}
	}
	return nil
}

// TriggerShutdown notifies all plugins that the server is shutting down.
func (r *Registry) TriggerShutdown() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		_ = p.OnShutdown()
	}
}

// List returns the names and versions of all registered plugins.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for _, p := range r.plugins {
		names = append(names, fmt.Sprintf("%s (v%s)", p.Name(), p.Version()))
	}
	return names
}
