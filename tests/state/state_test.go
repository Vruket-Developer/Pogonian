package state_test

import (
	"testing"
	"time"

	"pogonian/pkg/ir"
	"pogonian/pkg/state"
)

func TestStateEngine_TransferAndReconstruct(t *testing.T) {
	engine := state.NewEngine("test-secret-key-32-bytes-long!")

	// 1. Define App with UI and State Schemas
	app := &ir.Application{
		Name:    "multimedia-app",
		Version: "1.2.0",
		Routes: []*ir.RouteIR{
			{
				Path: "/player",
				UIRoot: &ir.UIElementIR{
					ID:   "video-player-container",
					Kind: ir.UIElementContainer,
					Children: []*ir.UIElementIR{
						{ID: "play-btn", Kind: ir.UIElementButton, Text: "Play"},
					},
				},
			},
		},
		State: &ir.StateIR{
			Fields: []*ir.StateFieldIR{
				{Name: "playback_position_sec", Type: "int", Scope: ir.StateScopeSync},
				{Name: "volume", Type: "int", Scope: ir.StateScopeSync},
				{Name: "ephemeral_hover_id", Type: "string", Scope: ir.StateScopeLocal}, // local: should not sync
			},
		},
	}

	// 2. Create state snapshot on Origin Platform (Web)
	webState := map[string]any{
		"playback_position_sec": 342,
		"volume":                85,
		"ephemeral_hover_id":    "item-4",
	}
	snapshot, err := engine.CreateSnapshot(app, ir.TargetWeb, "/player", "player-comp", webState)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// 3. Pack into tamper-evident Envelope
	envelopeBytes, err := engine.PackEnvelope(snapshot, 5*time.Minute)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}

	// 4. Unpack on Destination Platform (Android TV)
	recoveredSnapshot, err := engine.UnpackEnvelope(envelopeBytes)
	if err != nil {
		t.Fatalf("UnpackEnvelope failed: %v", err)
	}

	// 5. Reconstruct state on Android TV
	hydrated, err := engine.ReconstructState(recoveredSnapshot, app, ir.TargetAndroidTV)
	if err != nil {
		t.Fatalf("ReconstructState failed: %v", err)
	}

	// 6. Verify hydration accuracy
	if hydrated.OriginPlatform != ir.TargetWeb {
		t.Errorf("Expected origin Web, got %s", hydrated.OriginPlatform)
	}
	if hydrated.TargetPlatform != ir.TargetAndroidTV {
		t.Errorf("Expected target Android TV, got %s", hydrated.TargetPlatform)
	}
	posVal, ok := hydrated.ResolvedState["playback_position_sec"]
	if !ok || (posVal != 342 && posVal != float64(342)) {
		t.Errorf("Expected playback_position_sec 342, got %v (%T)", posVal, posVal)
	}
	// Verify local scoped field was correctly filtered out
	if _, ok := hydrated.ResolvedState["ephemeral_hover_id"]; ok {
		t.Errorf("Expected ephemeral_hover_id to be dropped due to local scope")
	}
	// Verify UI root was mapped
	if hydrated.TargetUIRoot == nil || hydrated.TargetUIRoot.ID != "video-player-container" {
		t.Errorf("Expected TargetUIRoot to be hydrated for route /player")
	}
}

func TestStateEngine_TamperRejection(t *testing.T) {
	engine := state.NewEngine("test-secret-key-32-bytes-long!")

	app := &ir.Application{
		Name:    "app",
		Version: "1.0.0",
	}

	snapshot, err := engine.CreateSnapshot(app, ir.TargetLinux, "/", "", map[string]any{"balance": 100})
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	envelope, err := ir.SignEnvelope(snapshot, "test-secret-key-32-bytes-long!", 10*time.Minute)
	if err != nil {
		t.Fatalf("SignEnvelope failed: %v", err)
	}

	// Tamper with state data
	envelope.Snapshot.StateData["balance"] = 999999

	// Verification must fail!
	err = envelope.Verify("test-secret-key-32-bytes-long!")
	if err == nil {
		t.Fatalf("Expected tamper verification error, but got nil!")
	}
}

func TestStateEngine_SecurityAndSemVer(t *testing.T) {
	// 1. Enforce no silent fallback key
	unsecuredEngine := state.NewEngine("")
	app := &ir.Application{Name: "app", Version: "1.9.0"}
	snapshot, err := unsecuredEngine.CreateSnapshot(app, ir.TargetWeb, "/", "", map[string]any{"data": "val"})
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	_, err = unsecuredEngine.PackEnvelope(snapshot, time.Minute)
	if err == nil {
		t.Fatalf("Expected error packing envelope without secret key, got nil!")
	}

	// 2. SemVer compatibility testing
	securedEngine := state.NewEngine("super-secret-key-1234567890123456")

	// 1.9.0 to 1.10.0 should be compatible (same major version 1)
	targetApp110 := &ir.Application{Name: "app", Version: "1.10.0", Routes: []*ir.RouteIR{{Path: "/"}}}
	if err := securedEngine.ValidateCompatibility(snapshot, targetApp110, ir.TargetAndroid); err != nil {
		t.Errorf("Expected 1.9.0 -> 1.10.0 to be compatible, got error: %v", err)
	}

	// 1.9.0 to 2.0.0 must FAIL (major version bump)
	targetApp200 := &ir.Application{Name: "app", Version: "2.0.0", Routes: []*ir.RouteIR{{Path: "/"}}}
	if err := securedEngine.ValidateCompatibility(snapshot, targetApp200, ir.TargetAndroid); err == nil {
		t.Errorf("Expected 1.9.0 -> 2.0.0 to fail major version compatibility, but it succeeded!")
	}

	// 3. Strict SemVer rejection of malformed versions (e.g., 1.hello.3)
	malformedApp := &ir.Application{Name: "app", Version: "1.hello.3", Routes: []*ir.RouteIR{{Path: "/"}}}
	if err := securedEngine.ValidateCompatibility(snapshot, malformedApp, ir.TargetAndroid); err == nil {
		t.Errorf("Expected malformed version '1.hello.3' to fail SemVer parsing, but it succeeded!")
	}

	malformedSnapshotApp := &ir.Application{Name: "app", Version: "1.2.world", Routes: []*ir.RouteIR{{Path: "/"}}}
	if err := securedEngine.ValidateCompatibility(snapshot, malformedSnapshotApp, ir.TargetAndroid); err == nil {
		t.Errorf("Expected malformed version '1.2.world' to fail SemVer parsing, but it succeeded!")
	}
}


