package state_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"pogonian/pkg/ir"
	"pogonian/pkg/state"
)

// TestStateMigration_Demonstration_WindowsToAndroid_OperationContinuation validates the end-to-end
// workflow requested by the user:
// Windows -> Open app -> Navigate /products/42 -> Modify state / start operation ->
// Generate migration state -> Transfer to Android -> Verify state + capabilities ->
// Reconstruct Android UI -> Continue from /products/42 and finish operation.
func TestStateMigration_Demonstration_WindowsToAndroid_OperationContinuation(t *testing.T) {
	engine := state.NewEngine("test-cryptographic-secret-key-32b!")
	opManager := state.NewOperationManager()

	// 1. Define Architecture-Driven Pogonian Application
	app := &ir.Application{
		Name:    "ecommerce-store",
		Version: "1.4.0",
		Routes: []*ir.RouteIR{
			{
				Path: "/products/:id",
				UIRoot: &ir.UIElementIR{
					ID:   "product-detail-view",
					Kind: ir.UIElementStackVertical,
					Children: []*ir.UIElementIR{
						{ID: "h-title", Kind: ir.UIElementHeading, Text: "Mechanical Keyboard 42"},
						{ID: "btn-checkout", Kind: ir.UIElementButton, Text: "Authorize & Purchase"},
						{ID: "txt-status", Kind: ir.UIElementText, Text: "Ready to checkout"},
					},
				},
				Params: []string{"id"},
			},
		},
		Entities: []*ir.EntityIR{
			{
				Name:      "Product",
				TableName: "products",
				Fields: []ir.FieldIR{
					{Name: "id", Type: "int64", PK: true},
					{Name: "name", Type: "string"},
					{Name: "price", Type: "float64"},
				},
			},
		},
		Repositories: []*ir.RepositoryIR{
			{Name: "ProductRepository", TargetTable: "products"},
		},
		Services: []*ir.ServiceIR{
			{Name: "ProductService", CallsRepos: []string{"ProductRepository"}},
		},
		Handlers: []*ir.HandlerIR{
			{Name: "ProductHandler", CallsServices: []string{"ProductService"}},
		},
		State: &ir.StateIR{
			Fields: []*ir.StateFieldIR{
				{Name: "cart_qty", Type: "int", Scope: ir.StateScopeSync},
				{Name: "order_notes", Type: "string", Scope: ir.StateScopeSync},
				{Name: "hovered_tooltip", Type: "string", Scope: ir.StateScopeLocal}, // local: should be dropped!
			},
		},
		Capabilities: []*ir.CapabilityIR{
			{Type: ir.CapabilityKeyboard, Required: false},
			{Type: ir.CapabilityTouch, Required: false},
			{Type: ir.CapabilityNotifications, Required: false},
		},
	}

	// Attach authoritative Mount contract
	prodMount := &ir.MountIR{
		ID:           "mount:resource:products",
		Resource:     "products",
		RoutePath:    "/products/:id",
		Entity:       "Product",
		Table:        "products",
		Repository:   "ProductRepository",
		Service:      "ProductService",
		Handler:      "ProductHandler",
		PrimaryKey:   "id",
		ParamBinding: "id",
	}
	app.Mounts = []*ir.MountIR{prodMount}
	app.Routes[0].Mount = prodMount

	// =========================================================================
	// Step 1: Running on Windows Desktop
	// =========================================================================
	t.Log("Step 1: Running on Windows Desktop at /products/42...")
	windowsState := map[string]any{
		"cart_qty":        2,
		"order_notes":     "Rush overnight shipping",
		"hovered_tooltip": "tooltip-active-item-42", // local ephemeral state
	}

	// User starts an active in-flight operation (e.g. checkout order)
	activeOp := opManager.StartOperation("checkout_order", "/api/products/42/checkout", map[string]any{
		"product_id": 42,
		"qty":        2,
		"total_usd":  299.98,
	})

	// Operation progresses through step 1 on Windows and checkpoints
	err := opManager.Checkpoint(activeOp.ID, 0.40, map[string]any{
		"step":          "payment_authorized",
		"payment_token": "pay_tok_secure_8831",
	})
	if err != nil {
		t.Fatalf("Failed to checkpoint operation: %v", err)
	}

	// =========================================================================
	// Step 2: Capture Migration Snapshot (with Partial State Slicing)
	// =========================================================================
	t.Log("Step 2: Capturing snapshot with active operations and partial state filtering...")
	activeOps := opManager.GetActiveOperations()
	if len(activeOps) != 1 {
		t.Fatalf("Expected 1 active operation, got %d", len(activeOps))
	}

	snapshot, err := engine.CreateMigrationSnapshot(
		app,
		ir.TargetWindows,
		"/products/42",
		"product-detail-view",
		windowsState,
		activeOps,
	)
	if err != nil {
		t.Fatalf("CreateMigrationSnapshot failed: %v", err)
	}

	// Verify partial state filtering
	if _, ok := snapshot.StateData["hovered_tooltip"]; ok {
		t.Errorf("Expected local scope 'hovered_tooltip' to be sliced out of migration payload")
	}
	if snapshot.StateData["cart_qty"] != 2 {
		t.Errorf("Expected cart_qty 2 in filtered state, got %v", snapshot.StateData["cart_qty"])
	}
	if snapshot.MountedResource == nil || snapshot.MountedResource.Resource != "products" {
		t.Errorf("Expected snapshot to capture mounted resource 'products'")
	}

	// =========================================================================
	// Step 3: Cryptographic Packaging & Transport
	// =========================================================================
	t.Log("Step 3: Packing tamper-evident envelope and transferring to Android...")
	envelopeBytes, err := engine.PackEnvelope(snapshot, 5*time.Minute)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}

	// =========================================================================
	// Step 4: Unpack & Verify on Destination (Android Mobile)
	// =========================================================================
	t.Log("Step 4: Unpacking envelope and verifying signature on Android...")
	recoveredSnapshot, err := engine.UnpackEnvelope(envelopeBytes)
	if err != nil {
		t.Fatalf("UnpackEnvelope failed: %v", err)
	}

	// =========================================================================
	// Step 5: Mount-Aware Pre-Flight & Capability-Aware Reconstruction on Android
	// =========================================================================
	t.Log("Step 5: Executing mount-aware reconstruction and continuing active operation on Android...")
	hydrated, err := engine.ReconstructState(recoveredSnapshot, app, ir.TargetAndroid)
	if err != nil {
		t.Fatalf("ReconstructState on Android failed: %v", err)
	}

	// Verify Mount Context Reconnected
	if hydrated.MountedResource == nil {
		t.Fatalf("Expected MountedResource to be restored on Android")
	}
	if hydrated.MountedResource.Entity != "Product" || hydrated.MountedResource.Table != "products" {
		t.Errorf("Expected mount entity 'Product' and table 'products', got %s / %s",
			hydrated.MountedResource.Entity, hydrated.MountedResource.Table)
	}

	// Verify Capability Adaptation
	if hydrated.AdaptationPlan == nil {
		t.Fatalf("Expected CapabilityAdaptationPlan to be populated")
	}
	if hydrated.AdaptationPlan.InputMode != "direct_touch" {
		t.Errorf("Expected Android input mode 'direct_touch', got '%s'", hydrated.AdaptationPlan.InputMode)
	}

	// Verify UI Adaptation
	if hydrated.TargetUIRoot == nil {
		t.Fatalf("Expected TargetUIRoot to be reconstructed for /products/42")
	}
	btnNode := hydrated.TargetUIRoot.Children[1]
	if btnNode.Properties["touch_target_dp"] != "48" {
		t.Errorf("Expected touch_target_dp '48' adapted on Android button, got '%v'", btnNode.Properties["touch_target_dp"])
	}

	// Verify Operation Continuation
	if len(hydrated.OperationResults) != 1 {
		t.Fatalf("Expected 1 operation result from continuation, got %d", len(hydrated.OperationResults))
	}
	opRes := hydrated.OperationResults[0]
	if opRes.Status != ir.OpStatusCompleted {
		t.Errorf("Expected continued operation status 'completed', got '%s'", opRes.Status)
	}
	if opRes.Progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", opRes.Progress)
	}
	if opRes.ResultData["resumed_from_step"] != "payment_authorized" {
		t.Errorf("Expected resumed_from_step 'payment_authorized', got '%v'", opRes.ResultData["resumed_from_step"])
	}
	if opRes.ResultData["resumed_on_platform"] != ir.TargetAndroid {
		t.Errorf("Expected resumed_on_platform 'android', got '%v'", opRes.ResultData["resumed_on_platform"])
	}

	t.Logf("✓ Active operation '%s' successfully continued and completed on Android!", opRes.OperationID)
	t.Logf("✓ Application state and UI successfully reconstructed for route %s", hydrated.ActiveRoute)
}

// TestStateMigration_MountAware_IncompatibilityRejection verifies that Pogonian refuses migration
// before corrupting state if the destination lacks the required data entity / repository.
func TestStateMigration_MountAware_IncompatibilityRejection(t *testing.T) {
	engine := state.NewEngine("test-secret-key-32-bytes-long!")

	originApp := &ir.Application{
		Name:    "store",
		Version: "1.0.0",
		Routes:  []*ir.RouteIR{{Path: "/admin/finance"}},
		Entities: []*ir.EntityIR{
			{Name: "Ledger", TableName: "ledgers"},
		},
		Mounts: []*ir.MountIR{
			{
				ID:        "mount:resource:finance",
				Resource:  "finance",
				RoutePath: "/admin/finance",
				Entity:    "Ledger",
				Table:     "ledgers",
			},
		},
	}

	// Destination app is a lightweight client missing Ledger entity
	destApp := &ir.Application{
		Name:     "store-client",
		Version:  "1.0.0",
		Routes:   []*ir.RouteIR{{Path: "/admin/finance"}},
		Entities: []*ir.EntityIR{}, // Ledger entity is missing!
	}

	snapshot, err := engine.CreateMigrationSnapshot(originApp, ir.TargetLinux, "/admin/finance", "", map[string]any{"balance": 5000}, nil)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Migration must be rejected during pre-flight mount check!
	_, err = engine.ReconstructState(snapshot, destApp, ir.TargetAndroid)
	if err == nil {
		t.Fatalf("Expected MountIncompatibilityError when migrating to destination lacking Ledger entity, got nil")
	}

	t.Logf("✓ Correctly rejected incompatible mount migration: %v", err)
}

// TestStateMigration_CapabilityAdaptation_AndroidTV verifies that migrating to Android TV
// adapts input mode to spatial D-pad and flags UI nodes as focusable.
func TestStateMigration_CapabilityAdaptation_AndroidTV(t *testing.T) {
	engine := state.NewEngine("test-secret-key-32-bytes-long!")

	app := &ir.Application{
		Name:    "media-app",
		Version: "1.0.0",
		Routes: []*ir.RouteIR{
			{
				Path: "/watch",
				UIRoot: &ir.UIElementIR{
					ID:   "player",
					Kind: ir.UIElementStackVertical,
					Children: []*ir.UIElementIR{
						{ID: "btn-play", Kind: ir.UIElementButton, Text: "Play"},
						{ID: "btn-settings", Kind: ir.UIElementButton, Text: "Settings"},
					},
				},
			},
		},
	}

	snapshot, err := engine.CreateMigrationSnapshot(app, ir.TargetWeb, "/watch", "player", map[string]any{"vol": 80}, nil)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	hydrated, err := engine.ReconstructState(snapshot, app, ir.TargetAndroidTV)
	if err != nil {
		t.Fatalf("ReconstructState for Android TV failed: %v", err)
	}

	if hydrated.AdaptationPlan.InputMode != "dpad_spatial" {
		t.Errorf("Expected TV input mode 'dpad_spatial', got '%s'", hydrated.AdaptationPlan.InputMode)
	}

	for _, child := range hydrated.TargetUIRoot.Children {
		if child.Properties["focusable"] != "true" {
			t.Errorf("Expected TV button %s to have focusable='true'", child.ID)
		}
	}
	t.Log("✓ Successfully verified Android TV spatial D-pad adaptation!")
}

// TestStateMigration_DeterministicConflictReconciliation verifies that simultaneous edits
// across offline devices reconcile deterministically using vector clocks.
func TestStateMigration_DeterministicConflictReconciliation(t *testing.T) {
	snapA := &ir.StateSnapshot{
		SnapshotID:  "snap_device_a",
		Timestamp:   1000,
		ActiveRoute: "/cart",
		StateData: map[string]any{
			"item_count": 3,
			"coupon":     "SAVE10",
		},
		StateVector: map[string]int64{
			"item_count": 1005, // Device A updated item_count later
			"coupon":     1000,
		},
	}

	snapB := &ir.StateSnapshot{
		SnapshotID:  "snap_device_b",
		Timestamp:   1002,
		ActiveRoute: "/cart",
		StateData: map[string]any{
			"item_count": 2,
			"coupon":     "SAVE20",
			"promo_flag": true,
		},
		StateVector: map[string]int64{
			"item_count": 1002,
			"coupon":     1010, // Device B updated coupon later
			"promo_flag": 1002,
		},
	}

	merged, resolutions := state.ReconcileStateConflicts(snapA, snapB)

	// Item count should be 3 (Device A vector clock 1005 > 1002)
	if merged.StateData["item_count"] != 3 {
		t.Errorf("Expected item_count 3 from Device A, got %v", merged.StateData["item_count"])
	}
	// Coupon should be SAVE20 (Device B vector clock 1010 > 1000)
	if merged.StateData["coupon"] != "SAVE20" {
		t.Errorf("Expected coupon 'SAVE20' from Device B, got %v", merged.StateData["coupon"])
	}
	// promo_flag was added by B
	if merged.StateData["promo_flag"] != true {
		t.Errorf("Expected promo_flag true, got %v", merged.StateData["promo_flag"])
	}

	t.Logf("✓ Successfully reconciled %d conflict resolutions deterministically!", len(resolutions))
}

// TestStateMigration_RealCapabilityEnforcement_CameraLocationOnTV tests the patent-critical
// capability checking scenario:
// Application requires: Camera, Location, Bluetooth
// Destination: Android TV
// Result: Camera ❌ (Unsupported), Location ❌ (Unsupported), Bluetooth ✅ (Supported)
// Migration: Reject if unhandled / Adapt according to declared fallback rules
func TestStateMigration_RealCapabilityEnforcement_CameraLocationOnTV(t *testing.T) {
	engine := state.NewEngine("test-cryptographic-secret-key-32b!")

	// 1. Scenario A: Strict Requirements Without Fallbacks -> Migration Must Reject
	appStrict := &ir.Application{
		Name:    "fleet-tracker",
		Version: "1.0.0",
		Capabilities: []*ir.CapabilityIR{
			{Type: ir.CapabilityCamera, Required: true},
			{Type: ir.CapabilityLocation, Required: true},
			{Type: ir.CapabilityBluetooth, Required: true},
		},
	}

	snapA := &ir.StateSnapshot{
		SnapshotID:           "snap-mobile-001",
		OriginPlatform:       ir.TargetAndroid,
		AppVersion:           "1.0.0",
		RequiredCapabilities: []ir.CapabilityType{ir.CapabilityCamera, ir.CapabilityLocation, ir.CapabilityBluetooth},
		StateData:            map[string]any{"device_mode": "tracking"},
	}

	envelopeStrict, err := engine.PackEnvelope(snapA, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to pack envelope: %v", err)
	}

	unpackedStrict, err := engine.UnpackEnvelope(envelopeStrict)
	if err != nil {
		t.Fatalf("Failed to unpack envelope: %v", err)
	}

	_, errStrict := engine.ReconstructState(unpackedStrict, appStrict, ir.TargetAndroidTV)
	if errStrict == nil {
		t.Fatalf("Expected migration to Android TV to be rejected due to unsupported Camera and Location, but it succeeded!")
	}
	t.Logf("✓ Scenario A: Successfully rejected unhandled migration to Android TV: %v", errStrict)

	// 2. Scenario B: Declared Fallback Policies -> Migration Adapts
	appWithFallbacks := &ir.Application{
		Name:    "fleet-tracker",
		Version: "1.0.0",
		Capabilities: []*ir.CapabilityIR{
			{Type: ir.CapabilityCamera, Required: false, FallbackAction: "use_static_device_avatar"},
			{Type: ir.CapabilityLocation, Required: false, FallbackAction: "use_manual_depot_selection"},
			{Type: ir.CapabilityBluetooth, Required: true},
		},
	}

	hydrated, errAdaptive := engine.ReconstructState(unpackedStrict, appWithFallbacks, ir.TargetAndroidTV)
	if errAdaptive != nil {
		t.Fatalf("Expected adaptive migration to Android TV to succeed with declared fallbacks, got error: %v", errAdaptive)
	}

	plan := hydrated.AdaptationPlan
	if plan == nil {
		t.Fatalf("Expected non-nil adaptation plan")
	}

	// Verify Bluetooth is supported
	var hasBluetooth bool
	for _, c := range plan.CompatibleCaps {
		if c == ir.CapabilityBluetooth {
			hasBluetooth = true
			break
		}
	}
	if !hasBluetooth {
		t.Errorf("Expected Bluetooth to be marked as Compatible on Android TV")
	}

	// Verify Camera and Location are degraded with declared fallback rules
	if plan.FallbackActions[ir.CapabilityCamera] != "use_static_device_avatar" {
		t.Errorf("Expected Camera fallback action 'use_static_device_avatar', got '%s'", plan.FallbackActions[ir.CapabilityCamera])
	}
	if plan.FallbackActions[ir.CapabilityLocation] != "use_manual_depot_selection" {
		t.Errorf("Expected Location fallback action 'use_manual_depot_selection', got '%s'", plan.FallbackActions[ir.CapabilityLocation])
	}

	t.Logf("✓ Scenario B: Successfully adapted on Android TV with Bluetooth ✅, Camera ❌ (Degraded with fallback), Location ❌ (Degraded with fallback)")
}

// =============================================================================
// PATENT-READINESS TEST SUITE (16 DEDICATED VERIFICATION SCENARIOS)
// =============================================================================

// Test 1: State capture with partial state slicing and checksum generation
func TestState_01_StateCapture(t *testing.T) {
	engine := state.NewEngine("secret-key-for-test-32-chars!!!")
	app := &ir.Application{
		Name:    "crm-app",
		Version: "1.0.0",
		State: &ir.StateIR{
			Fields: []*ir.StateFieldIR{
				{Name: "account_id", Type: "string", Scope: ir.StateScopeSync},
				{Name: "view_mode", Type: "string", Scope: ir.StateScopeSync},
				{Name: "popover_open", Type: "bool", Scope: ir.StateScopeLocal}, // Local: must be sliced
			},
		},
	}

	rawState := map[string]any{
		"account_id":   "acc_9921",
		"view_mode":    "grid",
		"popover_open": true,
	}

	snap, err := engine.CreateMigrationSnapshot(app, ir.TargetLinux, "/accounts", "grid-view", rawState, nil)
	if err != nil {
		t.Fatalf("CreateMigrationSnapshot failed: %v", err)
	}

	if snap.SnapshotID == "" {
		t.Errorf("Expected non-empty SnapshotID")
	}
	if snap.Nonce == "" {
		t.Errorf("Expected non-empty cryptographic Nonce")
	}
	if snap.SchemaChecksum == "" {
		t.Errorf("Expected non-empty SchemaChecksum")
	}
	if _, exists := snap.StateData["popover_open"]; exists {
		t.Errorf("Expected local scope 'popover_open' to be sliced out")
	}
	if snap.StateData["account_id"] != "acc_9921" || snap.StateData["view_mode"] != "grid" {
		t.Errorf("Expected sync fields preserved, got %v", snap.StateData)
	}
	t.Log("✓ Test 1 Passed: State capture and partial slicing verified")
}

// Test 2: State restoration into destination application
func TestState_02_StateRestoration(t *testing.T) {
	engine := state.NewEngine("secret-key-for-test-32-chars!!!")
	app := &ir.Application{
		Name:    "crm-app",
		Version: "1.0.0",
		Routes:  []*ir.RouteIR{{Path: "/accounts"}},
		State: &ir.StateIR{
			Fields: []*ir.StateFieldIR{
				{Name: "account_id", Type: "string", Scope: ir.StateScopeSync},
				{Name: "view_mode", Type: "string", Scope: ir.StateScopeSync},
			},
		},
	}

	snap := &ir.StateSnapshot{
		SnapshotID:     "snap-restore-01",
		Nonce:          "nonce-restore-01",
		AppVersion:     "1.0.0",
		OriginPlatform: ir.TargetWindows,
		ActiveRoute:    "/accounts",
		StateData: map[string]any{
			"account_id": "acc_9921",
			"view_mode":  "grid",
			"extra_prop": "ignored_value",
		},
	}

	hydrated, err := engine.ReconstructState(snap, app, ir.TargetAndroid)
	if err != nil {
		t.Fatalf("ReconstructState failed: %v", err)
	}

	if hydrated.ResolvedState["account_id"] != "acc_9921" {
		t.Errorf("Expected account_id acc_9921, got %v", hydrated.ResolvedState["account_id"])
	}
	if len(hydrated.RestoredFields) != 2 {
		t.Errorf("Expected 2 restored fields, got %d", len(hydrated.RestoredFields))
	}
	if len(hydrated.DroppedFields) != 1 {
		t.Errorf("Expected 1 dropped field for extra_prop, got %d", len(hydrated.DroppedFields))
	}
	t.Log("✓ Test 2 Passed: State restoration and field filtering verified")
}

// Test 3: Active operation checkpointing
func TestState_03_ActiveOperationCheckpoint(t *testing.T) {
	opManager := state.NewOperationManager()
	op := opManager.StartOperationWithStep(
		"invoice_export",
		"/api/invoices/export",
		"fetch_records",
		map[string]any{"format": "pdf", "range": "2026-Q1"},
		map[string]any{"worker_id": "worker_4"},
	)

	if op.CurrentStep != "fetch_records" || op.Progress != 0.0 {
		t.Errorf("Unexpected initial state: step=%s progress=%f", op.CurrentStep, op.Progress)
	}

	err := opManager.CheckpointStep(
		op.ID,
		"render_pdf_pages",
		0.65,
		map[string]any{"rendered_pages": 13, "total_pages": 20},
		map[string]any{"memory_mb": 42},
	)
	if err != nil {
		t.Fatalf("CheckpointStep failed: %v", err)
	}

	if op.CurrentStep != "render_pdf_pages" {
		t.Errorf("Expected step 'render_pdf_pages', got '%s'", op.CurrentStep)
	}
	if op.Progress != 0.65 {
		t.Errorf("Expected progress 0.65, got %f", op.Progress)
	}
	if op.Checkpoint["rendered_pages"] != 13 {
		t.Errorf("Expected rendered_pages 13, got %v", op.Checkpoint["rendered_pages"])
	}
	if op.ExecutionContext["memory_mb"] != 42 {
		t.Errorf("Expected memory_mb 42, got %v", op.ExecutionContext["memory_mb"])
	}
	t.Log("✓ Test 3 Passed: Active operation checkpointing verified")
}

// Test 4: Active operation continuation
func TestState_04_ActiveOperationContinuation(t *testing.T) {
	app := &ir.Application{Name: "billing-app", Version: "1.0.0"}
	op := &ir.ActiveOperationIR{
		ID:          "op_batch_01",
		Type:        "checkout_order",
		Target:      "/api/checkout",
		Status:      ir.OpStatusRunning,
		CurrentStep: "payment_authorized",
		Progress:    0.70,
		Payload:     map[string]any{"order_id": "ord_555", "amount": 149.99},
		Checkpoint:  map[string]any{"step": "payment_authorized", "auth_ref": "REF_9988"},
		Idempotency: "idem_token_abc123",
	}

	res, err := state.ContinueOperation(op, app, ir.TargetLinux)
	if err != nil {
		t.Fatalf("ContinueOperation failed: %v", err)
	}

	if res.Status != ir.OpStatusCompleted {
		t.Errorf("Expected status completed, got %s", res.Status)
	}
	if res.Progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", res.Progress)
	}
	if res.ResultData["resumed_from_step"] != "payment_authorized" {
		t.Errorf("Expected resumed_from_step 'payment_authorized', got %v", res.ResultData["resumed_from_step"])
	}
	if res.ResultData["idempotency_token"] != "idem_token_abc123" {
		t.Errorf("Expected idempotency token preserved, got %v", res.ResultData["idempotency_token"])
	}

	// Re-continuation must be idempotent and skip
	res2, err2 := state.ContinueOperation(op, app, ir.TargetLinux)
	if err2 != nil || res2.Status != ir.OpStatusCompleted {
		t.Errorf("Expected idempotent continuation to return completed immediately")
	}
	t.Log("✓ Test 4 Passed: Active operation continuation & idempotency verified")
}

// Test 5: Mount capture
func TestState_05_MountCapture(t *testing.T) {
	engine := state.NewEngine("secret-key-for-test-32-chars!!!")
	mount := &ir.MountIR{
		ID:         "mount:inventory",
		Resource:   "inventory",
		RoutePath:  "/warehouse/inventory/:item",
		Entity:     "InventoryItem",
		Table:      "inventory_items",
		Repository: "InventoryRepository",
		Service:    "InventoryService",
		Handler:    "InventoryHandler",
	}
	app := &ir.Application{
		Name:    "wms-app",
		Version: "1.0.0",
		Routes: []*ir.RouteIR{
			{Path: "/warehouse/inventory/:item", Mount: mount},
		},
		Mounts: []*ir.MountIR{mount},
	}

	snap, err := engine.CreateMigrationSnapshot(app, ir.TargetLinux, "/warehouse/inventory/SKU-99", "item-view", map[string]any{"sku": "SKU-99"}, nil)
	if err != nil {
		t.Fatalf("CreateMigrationSnapshot failed: %v", err)
	}

	if snap.MountedResource == nil {
		t.Fatalf("Expected MountedResource to be captured")
	}
	if snap.MountedResource.Resource != "inventory" {
		t.Errorf("Expected resource 'inventory', got '%s'", snap.MountedResource.Resource)
	}
	if snap.MountedResource.Entity != "InventoryItem" || snap.MountedResource.Table != "inventory_items" {
		t.Errorf("Expected entity 'InventoryItem' and table 'inventory_items'")
	}
	t.Log("✓ Test 5 Passed: Mount graph capture verified")
}

// Test 6: Mount validation
func TestState_06_MountValidation(t *testing.T) {
	mount := &ir.MountIR{
		ID:         "mount:inventory",
		Resource:   "inventory",
		RoutePath:  "/warehouse/inventory/:item",
		Entity:     "InventoryItem",
		Table:      "inventory_items",
		Repository: "InventoryRepository",
		Service:    "InventoryService",
	}
	destApp := &ir.Application{
		Name:    "wms-dest",
		Version: "1.0.0",
		Routes:  []*ir.RouteIR{{Path: "/warehouse/inventory/:item"}},
		Mounts:  []*ir.MountIR{mount},
		Entities: []*ir.EntityIR{
			{Name: "InventoryItem", TableName: "inventory_items"},
		},
		Repositories: []*ir.RepositoryIR{
			{Name: "InventoryRepository", TargetTable: "inventory_items"},
		},
		Services: []*ir.ServiceIR{
			{Name: "InventoryService"},
		},
	}

	snap := &ir.StateSnapshot{
		SnapshotID:      "snap-mount-val",
		AppVersion:      "1.0.0",
		ActiveRoute:     "/warehouse/inventory/SKU-99",
		MountedResource: mount,
	}

	err := state.PreflightMountCheck(snap, destApp, ir.TargetAndroid)
	if err != nil {
		t.Fatalf("PreflightMountCheck failed unexpectedly: %v", err)
	}
	t.Log("✓ Test 6 Passed: Mount validation verified")
}

// Test 7: Mount reconstruction & rejection on missing/incompatible contract
func TestState_07_MountReconstruction_And_Rejection(t *testing.T) {
	mount := &ir.MountIR{
		Resource:  "finance",
		RoutePath: "/finance/audit",
		Entity:    "AuditRecord",
		Table:     "audit_records",
	}
	snap := &ir.StateSnapshot{
		SnapshotID:      "snap-mount-recon",
		AppVersion:      "1.0.0",
		ActiveRoute:     "/finance/audit",
		MountedResource: mount,
	}

	// Target 1: Missing mount completely
	destAppNoMount := &ir.Application{
		Name:    "dest-no-mount",
		Version: "1.0.0",
		Routes:  []*ir.RouteIR{{Path: "/finance/audit"}},
	}
	_, err := state.ReconstructMountedResource(snap, destAppNoMount)
	if err == nil {
		t.Fatalf("Expected error for missing mount contract, got nil")
	}

	// Target 2: Mount exists but Entity does not match
	destAppMismatch := &ir.Application{
		Name:    "dest-mismatch",
		Version: "1.0.0",
		Routes:  []*ir.RouteIR{{Path: "/finance/audit"}},
		Mounts: []*ir.MountIR{
			{Resource: "finance", RoutePath: "/finance/audit", Entity: "DifferentEntity"},
		},
	}
	_, err = state.ReconstructMountedResource(snap, destAppMismatch)
	if err == nil {
		t.Fatalf("Expected error for mismatched entity in mount contract, got nil")
	}
	t.Log("✓ Test 7 Passed: Strict mount contract enforcement without auto-synthesis verified")
}

// Test 8: Capability detection (COMPATIBLE)
func TestState_08_CapabilityDetection_Compatible(t *testing.T) {
	app := &ir.Application{
		Name:    "desktop-utility",
		Version: "1.0.0",
		Capabilities: []*ir.CapabilityIR{
			{Type: ir.CapabilityFileSystem, Required: true},
			{Type: ir.CapabilityKeyboard, Required: true},
			{Type: ir.CapabilityMouse, Required: true},
		},
	}
	snap := &ir.StateSnapshot{
		SnapshotID:           "snap-cap-01",
		AppVersion:           "1.0.0",
		RequiredCapabilities: []ir.CapabilityType{ir.CapabilityFileSystem, ir.CapabilityKeyboard, ir.CapabilityMouse},
	}

	plan, err := state.AssessCapabilityAdaptation(snap, app, ir.TargetLinux)
	if err != nil {
		t.Fatalf("AssessCapabilityAdaptation failed: %v", err)
	}

	if plan.Status != state.StatusCompatible {
		t.Errorf("Expected status COMPATIBLE, got %s", plan.Status)
	}
	if len(plan.DegradedCaps) != 0 || len(plan.UnsupportedCaps) != 0 {
		t.Errorf("Expected zero degraded or unsupported caps, got %v / %v", plan.DegradedCaps, plan.UnsupportedCaps)
	}
	t.Log("✓ Test 8 Passed: Full capability compatibility detected")
}

// Test 9: Capability incompatibility (PARTIALLY_COMPATIBLE vs INCOMPATIBLE)
func TestState_09_CapabilityIncompatibility_Tripartite(t *testing.T) {
	// Case A: Strict requirement without fallback -> INCOMPATIBLE
	appStrict := &ir.Application{
		Name:    "worker-app",
		Version: "1.0.0",
		Capabilities: []*ir.CapabilityIR{
			{Type: ir.CapabilityBackgroundExec, Required: true}, // No fallback, unsupported on Web
		},
	}
	snap := &ir.StateSnapshot{
		SnapshotID:           "snap-cap-02",
		AppVersion:           "1.0.0",
		RequiredCapabilities: []ir.CapabilityType{ir.CapabilityBackgroundExec},
	}

	planStrict, err := state.AssessCapabilityAdaptation(snap, appStrict, ir.TargetWeb)
	if err == nil {
		t.Fatalf("Expected error for unsupported background execution without fallback, got nil")
	}
	if planStrict != nil && planStrict.Status != state.StatusIncompatible {
		t.Errorf("Expected status INCOMPATIBLE, got %v", planStrict.Status)
	}

	// Case B: Capability declared with fallback -> PARTIALLY_COMPATIBLE
	appWithFallback := &ir.Application{
		Name:    "worker-app-adaptive",
		Version: "1.0.0",
		Capabilities: []*ir.CapabilityIR{
			{Type: ir.CapabilityBackgroundExec, Required: false, FallbackAction: "poll_server_on_resume"},
		},
	}
	planAdaptive, err := state.AssessCapabilityAdaptation(snap, appWithFallback, ir.TargetWeb)
	if err != nil {
		t.Fatalf("Expected adaptive migration to succeed with fallback, got: %v", err)
	}
	if planAdaptive.Status != state.StatusPartiallyCompatible {
		t.Errorf("Expected status PARTIALLY_COMPATIBLE, got %s", planAdaptive.Status)
	}
	if planAdaptive.FallbackActions[ir.CapabilityBackgroundExec] != "poll_server_on_resume" {
		t.Errorf("Expected fallback action 'poll_server_on_resume', got '%s'", planAdaptive.FallbackActions[ir.CapabilityBackgroundExec])
	}
	t.Log("✓ Test 9 Passed: Tripartite capability assessment (COMPATIBLE/PARTIALLY_COMPATIBLE/INCOMPATIBLE) verified")
}

// Test 10: Secure migration package
func TestState_10_SecureMigrationPackage(t *testing.T) {
	secretKey := "my-secret-signing-key-32-bytes!"
	engine := state.NewEngine(secretKey)

	snap := &ir.StateSnapshot{
		SnapshotID:     "snap-secure-pkg",
		Nonce:          "cryptographic-nonce-12345",
		AppVersion:     "1.0.0",
		OriginPlatform: ir.TargetLinux,
		StateData:      map[string]any{"session": "active_user_1"},
	}

	envelopeBytes, err := engine.PackEnvelope(snap, 10*time.Minute)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(envelopeBytes, &rawMap); err != nil {
		t.Fatalf("Failed to unmarshal envelope JSON: %v", err)
	}

	if rawMap["signature"] == "" {
		t.Errorf("Expected non-empty cryptographic HMAC signature")
	}
	if rawMap["expires_at"] == nil {
		t.Errorf("Expected non-empty expires_at timestamp")
	}

	unpackedSnap, err := engine.UnpackEnvelope(envelopeBytes)
	if err != nil {
		t.Fatalf("UnpackEnvelope failed: %v", err)
	}
	if unpackedSnap.SnapshotID != snap.SnapshotID {
		t.Errorf("Expected snapshot ID '%s', got '%s'", snap.SnapshotID, unpackedSnap.SnapshotID)
	}
	t.Log("✓ Test 10 Passed: Cryptographic packaging and signature verification verified")
}

// Test 11: Version compatibility (SemVer)
func TestState_11_VersionCompatibility(t *testing.T) {
	engine := state.NewEngine("test-secret-key-32-bytes-long!")
	targetApp := &ir.Application{
		Name:    "ecommerce",
		Version: "1.4.0",
	}

	// Case A: Patch diff (1.4.0 vs 1.4.2) -> Compatible
	snapPatch := &ir.StateSnapshot{AppVersion: "1.4.2"}
	if err := engine.ValidateCompatibility(snapPatch, targetApp, ir.TargetAndroid); err != nil {
		t.Errorf("Expected patch version compatibility, got error: %v", err)
	}

	// Case B: Minor diff (1.4.0 vs 1.2.0) -> Compatible
	snapMinor := &ir.StateSnapshot{AppVersion: "1.2.0"}
	if err := engine.ValidateCompatibility(snapMinor, targetApp, ir.TargetAndroid); err != nil {
		t.Errorf("Expected minor version compatibility, got error: %v", err)
	}

	// Case C: Major diff (1.4.0 vs 2.0.0) -> Incompatible, must reject
	snapMajor := &ir.StateSnapshot{AppVersion: "2.0.0"}
	err := engine.ValidateCompatibility(snapMajor, targetApp, ir.TargetAndroid)
	if err == nil {
		t.Fatalf("Expected error for major version mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "major version mismatch") {
		t.Errorf("Expected 'major version mismatch' error message, got '%v'", err)
	}
	t.Log("✓ Test 11 Passed: SemVer version compatibility checking verified")
}

// Test 12: Corrupted migration package
func TestState_12_CorruptedMigrationPackage(t *testing.T) {
	secretKey := "my-secret-signing-key-32-bytes!"
	engine := state.NewEngine(secretKey)

	snap := &ir.StateSnapshot{
		SnapshotID: "snap-tamper-test",
		Nonce:      "nonce-tamper-test",
		AppVersion: "1.0.0",
		StateData:  map[string]any{"balance": 1000},
	}

	envelopeBytes, err := engine.PackEnvelope(snap, 5*time.Minute)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}

	// Tamper with envelope data (change balance from 1000 to 9999)
	tamperedBytes := []byte(strings.Replace(string(envelopeBytes), "1000", "9999", 1))

	_, err = engine.UnpackEnvelope(tamperedBytes)
	if err == nil {
		t.Fatalf("Expected UnpackEnvelope to reject tampered payload, but verification succeeded!")
	}
	t.Logf("✓ Test 12 Passed: Tampered migration package successfully rejected: %v", err)
}

// Test 13: Unauthorized migration and Anti-replay
func TestState_13_UnauthorizedMigration_And_AntiReplay(t *testing.T) {
	keyAuthorized := "legitimate-secret-key-32-bytes!"
	keyAttacker := "attacker-secret-key-32-bytes!!!"

	engineAuth := state.NewEngine(keyAuthorized)
	engineAttacker := state.NewEngine(keyAttacker)

	snap := &ir.StateSnapshot{
		SnapshotID: "snap-auth-replay",
		Nonce:      "nonce-auth-replay",
		AppVersion: "1.0.0",
		StateData:  map[string]any{"token": "auth_tok_xyz"},
	}

	// 1. Wrong key rejection
	envelopeBytes, err := engineAttacker.PackEnvelope(snap, 5*time.Minute)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}
	_, err = engineAuth.UnpackEnvelope(envelopeBytes)
	if err == nil {
		t.Fatalf("Expected rejection of envelope signed by unauthorized key, got nil")
	}

	// 2. Expired envelope rejection
	envelopeAuthBytes, err := engineAuth.PackEnvelope(snap, -1*time.Second)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}
	_, err = engineAuth.UnpackEnvelope(envelopeAuthBytes)
	if err == nil {
		t.Fatalf("Expected rejection of expired envelope, got nil")
	}

	// 3. Anti-replay rejection
	freshSnap := &ir.StateSnapshot{
		SnapshotID: "snap-fresh-01",
		Nonce:      "nonce-fresh-01",
		AppVersion: "1.0.0",
		StateData:  map[string]any{"seq": 1},
	}
	freshEnvelope, err := engineAuth.PackEnvelope(freshSnap, 5*time.Minute)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}

	// First unpack must succeed
	_, err = engineAuth.UnpackEnvelope(freshEnvelope)
	if err != nil {
		t.Fatalf("Initial unpack should succeed, got: %v", err)
	}

	// Second unpack with identical snapshot & nonce must fail anti-replay!
	_, errReplay := engineAuth.UnpackEnvelope(freshEnvelope)
	if errReplay == nil {
		t.Fatalf("Expected anti-replay violation on duplicate envelope, but it succeeded!")
	}
	if !strings.Contains(errReplay.Error(), "anti-replay violation") {
		t.Errorf("Expected anti-replay error message, got: %v", errReplay)
	}
	t.Log("✓ Test 13 Passed: Unauthorized key, expired TTL, and duplicate replay rejected")
}

// Test 14: Destination reconstruction (UI adaptation across platforms)
func TestState_14_DestinationReconstruction(t *testing.T) {
	rootUI := &ir.UIElementIR{
		ID:   "checkout-actions",
		Kind: ir.UIElementStackHorizontal,
		Children: []*ir.UIElementIR{
			{ID: "btn-confirm", Kind: ir.UIElementButton, Text: "Confirm Order"},
			{ID: "btn-cancel", Kind: ir.UIElementButton, Text: "Cancel"},
		},
	}

	// Reconstruct for Android Mobile
	androidUI := state.AdaptUIForTargetPlatform(rootUI, ir.TargetAndroid)
	if androidUI == nil {
		t.Fatalf("Expected adapted UI root for Android")
	}
	for _, child := range androidUI.Children {
		if child.Properties["touch_target_dp"] != "48" {
			t.Errorf("Expected Android button %s to have touch_target_dp=48, got %s", child.ID, child.Properties["touch_target_dp"])
		}
	}

	// Reconstruct for Android TV
	tvUI := state.AdaptUIForTargetPlatform(rootUI, ir.TargetAndroidTV)
	if tvUI == nil {
		t.Fatalf("Expected adapted UI root for Android TV")
	}
	for _, child := range tvUI.Children {
		if child.Properties["focusable"] != "true" {
			t.Errorf("Expected TV button %s to have focusable=true, got %s", child.ID, child.Properties["focusable"])
		}
	}
	t.Log("✓ Test 14 Passed: Destination UI reconstruction and platform adaptation verified")
}

// Test 15: Real desktop to Android migration
func TestState_15_DesktopToAndroidMigration(t *testing.T) {
	engine := state.NewEngine("test-secret-key-32-bytes-long!")
	mount := &ir.MountIR{
		ID:         "mount:catalog",
		Resource:   "catalog",
		RoutePath:  "/catalog/:category",
		Entity:     "CatalogCategory",
		Table:      "catalog_categories",
		Repository: "CatalogRepository",
		Service:    "CatalogService",
	}

	app := &ir.Application{
		Name:    "store-app",
		Version: "1.0.0",
		Routes: []*ir.RouteIR{
			{
				Path:  "/catalog/:category",
				Mount: mount,
				UIRoot: &ir.UIElementIR{
					ID:   "cat-view",
					Kind: ir.UIElementStackVertical,
					Children: []*ir.UIElementIR{
						{ID: "btn-refresh", Kind: ir.UIElementButton, Text: "Refresh"},
					},
				},
			},
		},
		Mounts: []*ir.MountIR{mount},
		Entities: []*ir.EntityIR{
			{Name: "CatalogCategory", TableName: "catalog_categories"},
		},
		Repositories: []*ir.RepositoryIR{
			{Name: "CatalogRepository", TargetTable: "catalog_categories"},
		},
		Services: []*ir.ServiceIR{
			{Name: "CatalogService"},
		},
		State: &ir.StateIR{
			Fields: []*ir.StateFieldIR{
				{Name: "category_name", Type: "string", Scope: ir.StateScopeSync},
				{Name: "scroll_position", Type: "int", Scope: ir.StateScopeLocal},
			},
		},
	}

	// Desktop execution context
	desktopState := map[string]any{
		"category_name":   "electronics",
		"scroll_position": 840,
	}

	snap, err := engine.CreateMigrationSnapshot(app, ir.TargetLinux, "/catalog/electronics", "cat-view", desktopState, nil)
	if err != nil {
		t.Fatalf("CreateMigrationSnapshot failed: %v", err)
	}

	envelopeBytes, err := engine.PackEnvelope(snap, 5*time.Minute)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}

	recoveredSnap, err := engine.UnpackEnvelope(envelopeBytes)
	if err != nil {
		t.Fatalf("UnpackEnvelope failed: %v", err)
	}

	hydrated, err := engine.ReconstructState(recoveredSnap, app, ir.TargetAndroid)
	if err != nil {
		t.Fatalf("ReconstructState failed on Android: %v", err)
	}

	if hydrated.TargetPlatform != ir.TargetAndroid {
		t.Errorf("Expected target platform android, got %s", hydrated.TargetPlatform)
	}
	if hydrated.ResolvedState["category_name"] != "electronics" {
		t.Errorf("Expected category_name electronics, got %v", hydrated.ResolvedState["category_name"])
	}
	if _, ok := hydrated.ResolvedState["scroll_position"]; ok {
		t.Errorf("Expected local ephemeral scroll_position to be dropped")
	}
	if hydrated.MountedResource == nil || hydrated.MountedResource.Resource != "catalog" {
		t.Errorf("Expected mounted resource 'catalog' reconnected on Android")
	}
	t.Log("✓ Test 15 Passed: Real Desktop -> Android migration verified")
}

// Test 16: Operation continuation after migration
func TestState_16_OperationContinuationAfterMigration(t *testing.T) {
	engine := state.NewEngine("test-secret-key-32-bytes-long!")
	opManager := state.NewOperationManager()

	app := &ir.Application{
		Name:    "store-app",
		Version: "1.0.0",
		Routes:  []*ir.RouteIR{{Path: "/checkout"}},
	}

	// 1. Operation starts on Windows Desktop
	op := opManager.StartOperationWithStep(
		"checkout_order",
		"/checkout/submit",
		"inventory_reserved",
		map[string]any{"cart_id": "cart_883", "total": 199.50},
		map[string]any{"client_os": "windows"},
	)

	// Checkpoint at 70%
	err := opManager.CheckpointStep(
		op.ID,
		"payment_authorized",
		0.70,
		map[string]any{"auth_code": "AUTH_OK_9918", "step": "payment_authorized"},
		map[string]any{"latency_ms": 120},
	)
	if err != nil {
		t.Fatalf("CheckpointStep failed: %v", err)
	}

	// 2. Capture and migrate to Android
	snap, err := engine.CreateMigrationSnapshot(app, ir.TargetWindows, "/checkout", "checkout-screen", map[string]any{}, []*ir.ActiveOperationIR{op})
	if err != nil {
		t.Fatalf("CreateMigrationSnapshot failed: %v", err)
	}

	envelopeBytes, err := engine.PackEnvelope(snap, 5*time.Minute)
	if err != nil {
		t.Fatalf("PackEnvelope failed: %v", err)
	}

	unpacked, err := engine.UnpackEnvelope(envelopeBytes)
	if err != nil {
		t.Fatalf("UnpackEnvelope failed: %v", err)
	}

	// 3. Reconstruct on Android and verify continuation
	hydrated, err := engine.ReconstructState(unpacked, app, ir.TargetAndroid)
	if err != nil {
		t.Fatalf("ReconstructState failed: %v", err)
	}

	if len(hydrated.OperationResults) != 1 {
		t.Fatalf("Expected 1 operation result, got %d", len(hydrated.OperationResults))
	}

	res := hydrated.OperationResults[0]
	if res.Status != ir.OpStatusCompleted {
		t.Errorf("Expected completed status, got %s", res.Status)
	}
	if res.Progress != 1.0 {
		t.Errorf("Expected 1.0 progress, got %f", res.Progress)
	}
	if res.ResultData["resumed_from_step"] != "payment_authorized" {
		t.Errorf("Expected resumed_from_step 'payment_authorized', got %v", res.ResultData["resumed_from_step"])
	}
	if res.ResultData["resumed_on_platform"] != ir.TargetAndroid {
		t.Errorf("Expected resumed_on_platform 'android', got %v", res.ResultData["resumed_on_platform"])
	}
	t.Log("✓ Test 16 Passed: Operation continuation after migration verified")
}

