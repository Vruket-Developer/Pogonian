package compiler

import (
	"fmt"
	"sync"

	"pogonian/pkg/ir"
)

// LoweredArtifact represents the compilation output tailored for a specific native platform.
type LoweredArtifact struct {
	Target             ir.TargetPlatform `json:"target"`
	AppName            string            `json:"app_name"`
	EntrypointFile     string            `json:"entrypoint_file"`
	Files              map[string][]byte `json:"files"`
	Summary            string            `json:"summary"`
	NativeCapabilities []string          `json:"native_capabilities"`
}

// Lowerer defines the platform lowering contract for Pogonian targets.
type Lowerer interface {
	Target() ir.TargetPlatform
	Lower(app *ir.Application, projectDir string) (*LoweredArtifact, error)
}

var (
	lowerersMu sync.RWMutex
	lowerers   = make(map[ir.TargetPlatform]Lowerer)
)

// RegisterLowerer registers a platform lowerer in the compiler registry.
func RegisterLowerer(l Lowerer) {
	lowerersMu.Lock()
	defer lowerersMu.Unlock()
	lowerers[l.Target()] = l
}

// GetLowerer retrieves the registered lowerer for a target platform.
func GetLowerer(target ir.TargetPlatform) (Lowerer, error) {
	lowerersMu.RLock()
	defer lowerersMu.RUnlock()

	l, ok := lowerers[target]
	if !ok {
		return nil, fmt.Errorf("no compiler lowerer registered for target platform '%s'", target)
	}
	return l, nil
}

// AvailableTargets returns all platforms with registered lowerers.
func AvailableTargets() []ir.TargetPlatform {
	lowerersMu.RLock()
	defer lowerersMu.RUnlock()

	var targets []ir.TargetPlatform
	for t := range lowerers {
		targets = append(targets, t)
	}
	return targets
}
