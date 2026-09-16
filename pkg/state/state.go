package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"pogonian/pkg/ir"
)

// HydratedState represents the fully mapped and reconstructed application state on a destination target.
type HydratedState struct {
	SnapshotID       string                    `json:"snapshot_id"`
	OriginPlatform   ir.TargetPlatform         `json:"origin_platform"`
	TargetPlatform   ir.TargetPlatform         `json:"target_platform"`
	ActiveRoute      string                    `json:"active_route"`
	ResolvedState    map[string]any            `json:"resolved_state"`
	RestoredFields   []string                  `json:"restored_fields"`
	DroppedFields    []string                  `json:"dropped_fields,omitempty"`
	TargetUIRoot     *ir.UIElementIR           `json:"target_ui_root,omitempty"`
	MountedResource  *ir.MountIR               `json:"mounted_resource,omitempty"`
	ActiveOperations []*ir.ActiveOperationIR   `json:"active_operations,omitempty"`
	OperationResults []*OperationResult        `json:"operation_results,omitempty"`
	AdaptationPlan   *CapabilityAdaptationPlan `json:"adaptation_plan,omitempty"`
}

// Engine manages portable state snapshots, cryptographic signing, compatibility checks, and reconstruction.
type Engine struct {
	secretKey   string
	replayMu    sync.RWMutex
	replayCache map[string]int64
}

// NewEngine creates a new cross-device state engine with a cryptographic signing key.
// If secretKey is empty, it attempts to load APP_KEY or SESSION_SECRET from environment.
func NewEngine(secretKey string) *Engine {
	if secretKey == "" {
		secretKey = os.Getenv("APP_KEY")
		if secretKey == "" {
			secretKey = os.Getenv("SESSION_SECRET")
		}
	}
	return &Engine{
		secretKey:   secretKey,
		replayCache: make(map[string]int64),
	}
}

// CreateSnapshot captures the current application state for cross-device transfer.
func (e *Engine) CreateSnapshot(app *ir.Application, origin ir.TargetPlatform, activeRoute string, activeComp string, stateData map[string]any) (*ir.StateSnapshot, error) {
	return e.CreateMigrationSnapshot(app, origin, activeRoute, activeComp, stateData, nil)
}

// CreateMigrationSnapshot captures state, active operations, mount context, and vector clocks with partial state slicing.
func (e *Engine) CreateMigrationSnapshot(app *ir.Application, origin ir.TargetPlatform, activeRoute string, activeComp string, stateData map[string]any, activeOps []*ir.ActiveOperationIR) (*ir.StateSnapshot, error) {
	snapID := generateRandomID(16)
	now := time.Now().Unix()

	// 1. Partial State Slicing: Filter out local component scopes
	filteredState := make(map[string]any)
	stateVector := make(map[string]int64)

	var knownFields map[string]*ir.StateFieldIR
	if app.State != nil {
		knownFields = make(map[string]*ir.StateFieldIR)
		for _, f := range app.State.Fields {
			knownFields[f.Name] = f
		}
	}

	for k, v := range stateData {
		if knownFields != nil {
			if fDef, ok := knownFields[k]; ok {
				if fDef.Scope == ir.StateScopeLocal {
					continue // slice out local ephemeral state
				}
			}
		}
		filteredState[k] = v
		stateVector[k] = now
	}

	// 2. Resolve Active Mount
	var mountedRes *ir.MountIR
	if activeRoute != "" {
		for _, m := range app.Mounts {
			if matchRoute(m.RoutePath, activeRoute) {
				mountedRes = m
				break
			}
		}
		if mountedRes == nil {
			mountedRes = app.FindMountByRoute(activeRoute)
		}
		if mountedRes == nil && len(app.Mounts) > 0 {
			cleanPath := strings.Trim(activeRoute, "/")
			parts := strings.Split(cleanPath, "/")
			for _, part := range parts {
				if m := app.FindMountByResource(part); m != nil {
					mountedRes = m
					break
				}
			}
		}
	}

	// 3. Extract Required Capabilities
	var reqCaps []ir.CapabilityType
	for _, c := range app.Capabilities {
		if c.Required {
			reqCaps = append(reqCaps, c.Type)
		}
	}

	snapshot := &ir.StateSnapshot{
		SnapshotID:           snapID,
		Nonce:                generateRandomID(16),
		Timestamp:            now,
		AppVersion:           app.Version,
		OriginPlatform:       origin,
		ActiveRoute:          activeRoute,
		ActiveComponentID:    activeComp,
		StateData:            filteredState,
		ActiveOperations:     activeOps,
		MountedResource:      mountedRes,
		VersionClock:         1,
		StateVector:          stateVector,
		RequiredCapabilities: reqCaps,
	}

	chk, err := snapshot.ComputeChecksum()
	if err != nil {
		return nil, fmt.Errorf("failed to compute state snapshot checksum: %w", err)
	}
	snapshot.SchemaChecksum = chk

	return snapshot, nil
}

// PackEnvelope packages and signs the snapshot into a tamper-proof JSON payload.
func (e *Engine) PackEnvelope(snapshot *ir.StateSnapshot, ttl time.Duration) ([]byte, error) {
	if e.secretKey == "" {
		return nil, fmt.Errorf("state engine signing error: no secure signing key configured (must set APP_KEY or SESSION_SECRET)")
	}
	envelope, err := ir.SignEnvelope(snapshot, e.secretKey, ttl)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope)
}

// checkAndRecordReplay ensures the incoming snapshot and nonce have not been previously hydrated within TTL.
func (e *Engine) checkAndRecordReplay(envelope *ir.StateEnvelope) error {
	if envelope == nil || envelope.Snapshot == nil {
		return fmt.Errorf("invalid envelope for replay check")
	}
	e.replayMu.Lock()
	defer e.replayMu.Unlock()

	now := time.Now().Unix()

	// Purge expired replay entries
	for k, exp := range e.replayCache {
		if now > exp {
			delete(e.replayCache, k)
		}
	}

	key := envelope.Snapshot.SnapshotID
	if envelope.Snapshot.Nonce != "" {
		key += ":" + envelope.Snapshot.Nonce
	}

	if exp, exists := e.replayCache[key]; exists && now <= exp {
		return fmt.Errorf("anti-replay violation: snapshot '%s' has already been processed and cannot be replayed", envelope.Snapshot.SnapshotID)
	}

	e.replayCache[key] = envelope.ExpiresAt
	return nil
}

// UnpackEnvelope parses and cryptographically verifies the state envelope.
func (e *Engine) UnpackEnvelope(envelopeBytes []byte) (*ir.StateSnapshot, error) {
	if e.secretKey == "" {
		return nil, fmt.Errorf("state engine verification error: no secure signing key configured (must set APP_KEY or SESSION_SECRET)")
	}
	var envelope ir.StateEnvelope
	if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse state envelope JSON: %w", err)
	}

	if err := envelope.Verify(e.secretKey); err != nil {
		return nil, fmt.Errorf("state envelope verification failed: %w", err)
	}

	if err := e.checkAndRecordReplay(&envelope); err != nil {
		return nil, err
	}

	return envelope.Snapshot, nil
}

// ValidateCompatibility checks whether the incoming state snapshot is compatible with the destination application.
func (e *Engine) ValidateCompatibility(snapshot *ir.StateSnapshot, targetApp *ir.Application, targetPlatform ir.TargetPlatform) error {
	if snapshot.AppVersion != targetApp.Version {
		// Minor/patch versions can continue with warnings; major version mismatch is an error
		if !isMajorVersionCompatible(snapshot.AppVersion, targetApp.Version) {
			return fmt.Errorf("major version mismatch: snapshot v%s incompatible with target v%s",
				snapshot.AppVersion, targetApp.Version)
		}
	}

	// Verify route exists on destination application (supporting parameterized paths)
	if snapshot.ActiveRoute != "" {
		routeFound := false
		for _, r := range targetApp.Routes {
			if matchRoute(r.Path, snapshot.ActiveRoute) {
				routeFound = true
				break
			}
		}
		if !routeFound && len(targetApp.Routes) > 0 {
			return fmt.Errorf("active route '%s' from snapshot does not exist in target application", snapshot.ActiveRoute)
		}
	}

	return nil
}

func matchRoute(routePattern, path string) bool {
	if routePattern == path {
		return true
	}
	p1 := strings.Split(strings.Trim(routePattern, "/"), "/")
	p2 := strings.Split(strings.Trim(path, "/"), "/")
	if len(p1) != len(p2) {
		return false
	}
	for i := range p1 {
		if strings.HasPrefix(p1[i], ":") || (strings.HasPrefix(p1[i], "[") && strings.HasSuffix(p1[i], "]")) {
			continue // matches dynamic parameter
		}
		if p1[i] != p2[i] {
			return false
		}
	}
	return true
}

// ReconstructState reconciles portable state data into the target platform application model and active UI tree.
func (e *Engine) ReconstructState(snapshot *ir.StateSnapshot, targetApp *ir.Application, targetPlatform ir.TargetPlatform) (*HydratedState, error) {
	// 1. Compatibility Check (SemVer and route)
	if err := e.ValidateCompatibility(snapshot, targetApp, targetPlatform); err != nil {
		return nil, fmt.Errorf("cannot reconstruct state: %w", err)
	}

	// 2. Pre-flight Mount Verification
	if err := PreflightMountCheck(snapshot, targetApp, targetPlatform); err != nil {
		return nil, err
	}

	// 3. Hardware Capability Adaptation
	adaptationPlan, err := AssessCapabilityAdaptation(snapshot, targetApp, targetPlatform)
	if err != nil {
		return nil, fmt.Errorf("capability adaptation failed: %w", err)
	}

	// 4. Reconstruct Mounted Resource
	reconstructedMount, err := ReconstructMountedResource(snapshot, targetApp)
	if err != nil {
		return nil, fmt.Errorf("mount reconstruction failed: %w", err)
	}

	// 5. Restore State Fields
	resolvedState := make(map[string]any)
	var restoredFields []string
	var droppedFields []string

	var knownFields map[string]*ir.StateFieldIR
	if targetApp.State != nil {
		knownFields = make(map[string]*ir.StateFieldIR)
		for _, f := range targetApp.State.Fields {
			knownFields[f.Name] = f
		}
	}

	for k, v := range snapshot.StateData {
		if knownFields != nil {
			if fieldDef, ok := knownFields[k]; ok {
				if fieldDef.Scope == ir.StateScopeLocal {
					droppedFields = append(droppedFields, fmt.Sprintf("%s (local scope not synced)", k))
					continue
				}
				resolvedState[k] = v
				restoredFields = append(restoredFields, k)
			} else {
				droppedFields = append(droppedFields, fmt.Sprintf("%s (unknown field)", k))
			}
		} else {
			resolvedState[k] = v
			restoredFields = append(restoredFields, k)
		}
	}

	// 6. Reconstruct and Adapt UI Tree for Target Platform
	var targetUIRoot *ir.UIElementIR
	for _, r := range targetApp.Routes {
		if matchRoute(r.Path, snapshot.ActiveRoute) {
			targetUIRoot = AdaptUIForTargetPlatform(r.UIRoot, targetPlatform)
			break
		}
	}

	// 7. Continue Active In-Flight Operations
	var opResults []*OperationResult
	for _, op := range snapshot.ActiveOperations {
		res, err := ContinueOperation(op, targetApp, targetPlatform)
		if err != nil {
			opResults = append(opResults, &OperationResult{
				OperationID: op.ID,
				Type:        op.Type,
				Status:      ir.OpStatusFailed,
				Message:     err.Error(),
			})
		} else {
			opResults = append(opResults, res)
		}
	}

	return &HydratedState{
		SnapshotID:       snapshot.SnapshotID,
		OriginPlatform:   snapshot.OriginPlatform,
		TargetPlatform:   targetPlatform,
		ActiveRoute:      snapshot.ActiveRoute,
		ResolvedState:    resolvedState,
		RestoredFields:   restoredFields,
		DroppedFields:    droppedFields,
		TargetUIRoot:     targetUIRoot,
		MountedResource:  reconstructedMount,
		ActiveOperations: snapshot.ActiveOperations,
		OperationResults: opResults,
		AdaptationPlan:   adaptationPlan,
	}, nil
}

// SemVer represents parsed semantic version components (Major.Minor.Patch).
type SemVer struct {
	Major int
	Minor int
	Patch int
}

// ParseSemVer parses semantic version string into numerical components.
func ParseSemVer(v string) (SemVer, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) == 0 || parts[0] == "" {
		return SemVer{}, fmt.Errorf("empty version string")
	}

	var sv SemVer
	maj, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major version '%s': %w", parts[0], err)
	}
	sv.Major = maj

	if len(parts) > 1 {
		if parts[1] == "" {
			return SemVer{}, fmt.Errorf("empty minor version component")
		}
		min, err := strconv.Atoi(parts[1])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid minor version '%s': %w", parts[1], err)
		}
		sv.Minor = min
	}
	if len(parts) > 2 {
		if parts[2] == "" {
			return SemVer{}, fmt.Errorf("empty patch version component")
		}
		patchStr := strings.Split(parts[2], "-")[0]
		patch, err := strconv.Atoi(patchStr)
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid patch version '%s': %w", patchStr, err)
		}
		sv.Patch = patch
	}
	return sv, nil
}

func isMajorVersionCompatible(v1, v2 string) bool {
	sv1, err1 := ParseSemVer(v1)
	sv2, err2 := ParseSemVer(v2)
	if err1 != nil || err2 != nil {
		return false
	}
	return sv1.Major == sv2.Major
}

func generateRandomID(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
