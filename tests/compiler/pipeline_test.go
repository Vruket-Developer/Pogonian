package compiler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/pkg/compiler"
	"pogonian/pkg/ir"
)

func TestCompilerPipeline_MultiTargetLowering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogn-compiler-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create minimal app structure
	_ = os.MkdirAll(filepath.Join(tempDir, "app"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "app", "page.html"), []byte("<h1>Home</h1>"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "pogonian.yaml"), []byte("name: test-app\nversion: 1.0.0\n"), 0644)

	pipeline, err := compiler.NewPipeline(tempDir)
	if err != nil {
		t.Fatalf("NewPipeline failed: %v", err)
	}

	targets := []ir.TargetPlatform{
		ir.TargetWeb,
		ir.TargetAndroid,
		ir.TargetIOS,
		ir.TargetLinux,
		ir.TargetWindows,
		ir.TargetMacOS,
		ir.TargetAndroidTV,
	}

	res, err := pipeline.Compile(targets)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	if len(res.Artifacts) != len(targets) {
		t.Errorf("Expected %d lowered artifacts, got %d", len(targets), len(res.Artifacts))
	}

	// Verify Web artifact
	webArt, ok := res.Artifacts[ir.TargetWeb]
	if !ok || len(webArt.Files["public/runtime.js"]) == 0 {
		t.Errorf("Web artifact missing runtime.js")
	}

	// Verify Android artifact (Compose)
	androidArt, ok := res.Artifacts[ir.TargetAndroid]
	if !ok {
		t.Errorf("Android artifact missing")
	} else {
		var foundRootView bool
		for path := range androidArt.Files {
			if strings.HasSuffix(path, "PogonianRootView.kt") {
				foundRootView = true
				break
			}
		}
		if !foundRootView {
			t.Errorf("Android artifact missing Compose root view")
		}
	}

	// Verify iOS artifact (SwiftUI)
	iosArt, ok := res.Artifacts[ir.TargetIOS]
	if !ok || len(iosArt.Files["ios/Views/PogonianRootView.swift"]) == 0 {
		t.Errorf("iOS artifact missing SwiftUI root view")
	}

	// Verify Android TV artifact (10-foot D-Pad Compose)
	tvArt, ok := res.Artifacts[ir.TargetAndroidTV]
	if !ok {
		t.Errorf("Android TV artifact missing")
	} else {
		var foundTVRootView bool
		for path := range tvArt.Files {
			if strings.HasSuffix(path, "PogonianTVRootView.kt") {
				foundTVRootView = true
				break
			}
		}
		if !foundTVRootView {
			t.Errorf("Android TV artifact missing TV root view")
		}
	}
}

func TestCompilerPipeline_DirectLoweringUIElements(t *testing.T) {
	app := &ir.Application{
		Name:    "store",
		Version: "1.0.0",
		Routes: []*ir.RouteIR{
			{
				Path: "/checkout",
				UIRoot: &ir.UIElementIR{
					ID:   "checkout-stack",
					Kind: ir.UIElementStackVertical,
					Children: []*ir.UIElementIR{
						{ID: "h1", Kind: ir.UIElementHeading, Text: "Order Summary"},
						{ID: "btn", Kind: ir.UIElementButton, Text: "Pay Now"},
					},
				},
			},
		},
	}

	// Lower to Web
	webLowerer, err := compiler.GetLowerer(ir.TargetWeb)
	if err != nil {
		t.Fatalf("GetLowerer Web failed: %v", err)
	}
	webArt, err := webLowerer.Lower(app, ".")
	if err != nil {
		t.Fatalf("Web lower failed: %v", err)
	}
	if len(webArt.Files["app/checkout.html"]) == 0 {
		t.Errorf("Expected checkout.html in Web files")
	}

	// Lower to Android
	androidLowerer, err := compiler.GetLowerer(ir.TargetAndroid)
	if err != nil {
		t.Fatalf("GetLowerer Android failed: %v", err)
	}
	androidArt, err := androidLowerer.Lower(app, ".")
	if err != nil {
		t.Fatalf("Android lower failed: %v", err)
	}
	var composeCode string
	for path, content := range androidArt.Files {
		if strings.HasSuffix(path, "PogonianRootView.kt") {
			composeCode = string(content)
			break
		}
	}
	if len(composeCode) == 0 {
		t.Errorf("Expected compose code for Android")
	}

	// Lower to iOS
	iosLowerer, err := compiler.GetLowerer(ir.TargetIOS)
	if err != nil {
		t.Fatalf("GetLowerer iOS failed: %v", err)
	}
	iosArt, err := iosLowerer.Lower(app, ".")
	if err != nil {
		t.Fatalf("iOS lower failed: %v", err)
	}
	swiftCode := string(iosArt.Files["ios/Views/PogonianRootView.swift"])
	if len(swiftCode) == 0 {
		t.Errorf("Expected SwiftUI code for iOS")
	}
}

func TestCompilerPipeline_FullNativeProjectStructure(t *testing.T) {
	app := &ir.Application{
		Name:    "SuperApp",
		Version: "2.1.0",
		Capabilities: []*ir.CapabilityIR{
			{Type: ir.CapabilityCamera, Required: true},
			{Type: ir.CapabilityLocation, Required: false},
		},
		State: &ir.StateIR{
			Fields: []*ir.StateFieldIR{
				{Name: "user_name", Type: "string", InitialValue: "Alice", Scope: ir.StateScopeSync},
				{Name: "counter", Type: "int", InitialValue: 42, Scope: ir.StateScopeSync},
			},
		},
		Routes: []*ir.RouteIR{
			{
				Path: "/dashboard",
				UIRoot: &ir.UIElementIR{
					ID:   "dash-root",
					Kind: ir.UIElementStackVertical,
					Children: []*ir.UIElementIR{
						{ID: "h1", Kind: ir.UIElementHeading, Text: "Dashboard"},
						{
							ID:   "submit-btn",
							Kind: ir.UIElementButton,
							Text: "Save",
							Events: []*ir.EventIR{
								{
									Type: ir.EventClick,
									Actions: []*ir.ActionIR{
										{Type: ir.ActionMutateState, Target: "user_name"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// 1. Android Complete Project Test
	androidLowerer, err := compiler.GetLowerer(ir.TargetAndroid)
	if err != nil {
		t.Fatalf("GetLowerer Android failed: %v", err)
	}
	androidArt, err := androidLowerer.Lower(app, ".")
	if err != nil {
		t.Fatalf("Android lower failed: %v", err)
	}
	expectedAndroidFiles := []string{
		"android/build.gradle.kts",
		"android/settings.gradle.kts",
		"android/app/build.gradle.kts",
		"android/app/src/main/AndroidManifest.xml",
		"android/app/src/main/kotlin/com/pogonian/superapp/PogonianViewModel.kt",
		"android/app/src/main/kotlin/com/pogonian/superapp/MainActivity.kt",
		"android/app/src/main/kotlin/com/pogonian/superapp/ui/PogonianRootView.kt",
	}
	for _, f := range expectedAndroidFiles {
		if len(androidArt.Files[f]) == 0 {
			t.Errorf("Android project missing required file: %s", f)
		}
	}
	// Verify that duplicate com/pogonian/app leftover files were NOT generated
	if len(androidArt.Files["android/app/src/main/kotlin/com/pogonian/app/PogonianViewModel.kt"]) != 0 {
		t.Errorf("Unexpected duplicate leftover file: com/pogonian/app/PogonianViewModel.kt should not exist")
	}

	// Verify Android camera & location permissions
	manifestXML := string(androidArt.Files["android/app/src/main/AndroidManifest.xml"])
	if !strings.Contains(manifestXML, "android.permission.CAMERA") {
		t.Errorf("Expected AndroidManifest to contain CAMERA permission")
	}

	// Verify native Android capability invocations in PogonianViewModel
	vmCode := string(androidArt.Files["android/app/src/main/kotlin/com/pogonian/superapp/PogonianViewModel.kt"])
	if !strings.Contains(vmCode, "MediaStore.ACTION_IMAGE_CAPTURE") {
		t.Errorf("Expected PogonianViewModel to invoke MediaStore for camera capability")
	}
	if !strings.Contains(vmCode, "LocationManager") {
		t.Errorf("Expected PogonianViewModel to invoke LocationManager for location capability")
	}
	if !strings.Contains(vmCode, "BiometricManager") {
		t.Errorf("Expected PogonianViewModel to invoke BiometricManager for biometrics capability")
	}

	// 2. iOS Complete Project Test
	iosLowerer, err := compiler.GetLowerer(ir.TargetIOS)
	if err != nil {
		t.Fatalf("GetLowerer iOS failed: %v", err)
	}
	iosArt, err := iosLowerer.Lower(app, ".")
	if err != nil {
		t.Fatalf("iOS lower failed: %v", err)
	}
	expectedIOSFiles := []string{
		"ios/Package.swift",
		"ios/Info.plist",
		"ios/Sources/PogonianViewModel.swift",
		"ios/Sources/PogonianApp.swift",
		"ios/Sources/PogonianRootView.swift",
	}
	for _, f := range expectedIOSFiles {
		if len(iosArt.Files[f]) == 0 {
			t.Errorf("iOS project missing required file: %s", f)
		}
	}
	plistContent := string(iosArt.Files["ios/Info.plist"])
	if !strings.Contains(plistContent, "NSCameraUsageDescription") {
		t.Errorf("Expected Info.plist to contain NSCameraUsageDescription")
	}

	// 3. Desktop Complete Project Test (Linux, Windows, macOS)
	for _, plat := range []ir.TargetPlatform{ir.TargetLinux, ir.TargetWindows, ir.TargetMacOS} {
		deskLowerer, err := compiler.GetLowerer(plat)
		if err != nil {
			t.Fatalf("GetLowerer %s failed: %v", plat, err)
		}
		art, err := deskLowerer.Lower(app, ".")
		if err != nil {
			t.Fatalf("Desktop %s lower failed: %v", plat, err)
		}
		if len(art.Files[fmt.Sprintf("desktop/%s/go.mod", plat)]) == 0 {
			t.Errorf("Missing go.mod for desktop %s", plat)
		}
		mainCode := string(art.Files[fmt.Sprintf("desktop/%s/main.go", plat)])
		if !strings.Contains(mainCode, "NewAppState") || !strings.Contains(mainCode, "Dispatch") {
			t.Errorf("Desktop %s main.go missing reactive AppState or Dispatch", plat)
		}
	}

	// 4. Android TV Test
	tvLowerer, err := compiler.GetLowerer(ir.TargetAndroidTV)
	if err != nil {
		t.Fatalf("GetLowerer AndroidTV failed: %v", err)
	}
	tvArt, err := tvLowerer.Lower(app, ".")
	if err != nil {
		t.Fatalf("TV lower failed: %v", err)
	}
	tvManifest := string(tvArt.Files["android_tv/src/main/AndroidManifest.xml"])
	if !strings.Contains(tvManifest, "android.software.leanback") {
		t.Errorf("Expected Leanback feature in TV Manifest")
	}
}

func TestCompilerPipeline_MountSystem(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogn-mount-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	_ = os.MkdirAll(filepath.Join(tempDir, "app", "products", "[id]"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "app", "products", "page.html"), []byte("<h1>Products</h1>"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "app", "products", "[id]", "page.html"), []byte("<h1>Product Detail</h1>"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "pogonian.yaml"), []byte("name: shop\nversion: 1.0.0\n"), 0644)

	pipeline, err := compiler.NewPipeline(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	res, err := pipeline.Compile([]ir.TargetPlatform{ir.TargetWeb})
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	if len(res.AppIR.Mounts) < 2 {
		t.Fatalf("Expected at least 2 auto-wired mounts, got %d", len(res.AppIR.Mounts))
	}

	prodMount := res.AppIR.FindMountByRoute("/products")
	if prodMount == nil {
		t.Fatalf("Expected mount for /products")
	}
	if prodMount.Table != "products" {
		t.Errorf("Expected mount table to be 'products', got '%s'", prodMount.Table)
	}
	if prodMount.Entity != "Products" {
		t.Errorf("Expected mount entity to be 'Products', got '%s'", prodMount.Entity)
	}
}

func TestCompilerPipeline_ArchitectureDrivenMountsAndValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogonian-mount-arch-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create UI route with parameter
	_ = os.MkdirAll(filepath.Join(tempDir, "app", "items", "[id]"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "app", "items", "[id]", "page.html"), []byte("<h1>Item</h1>"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "pogonian.yaml"), []byte("name: inventory\nversion: 1.0.0\n"), 0644)

	// Create backend server architecture with Entity, Repo, Service, Handler
	_ = os.MkdirAll(filepath.Join(tempDir, "server", "items"), 0755)
	entityCode := `package items

type Item struct {
	ID    int64  ` + "`json:\"id\"`" + `
	Title string ` + "`json:\"title\"`" + `
}

type ItemRepository struct {}

type ItemService struct {
	repo *ItemRepository
}

type ItemHandler struct {
	svc *ItemService
}
`
	_ = os.WriteFile(filepath.Join(tempDir, "server", "items", "item.go"), []byte(entityCode), 0644)

	pipeline, err := compiler.NewPipeline(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	res, err := pipeline.Compile([]ir.TargetPlatform{ir.TargetWeb})
	if err != nil {
		t.Fatalf("Architecture-driven compilation failed: %v", err)
	}

	// 1. Authoritative Resource Mount
	resMount := res.AppIR.FindMountByResource("items")
	if resMount == nil {
		t.Fatalf("Expected authoritative resource mount for 'items'")
	}
	if resMount.Entity != "Item" {
		t.Errorf("Expected mount entity 'Item', got '%s'", resMount.Entity)
	}
	if resMount.Table != "items" {
		t.Errorf("Expected mount table 'items', got '%s'", resMount.Table)
	}

	// 2. Route Mount with PrimaryKey binding
	routeMount := res.AppIR.FindMountByRoute("/items/[id]")
	if routeMount == nil {
		t.Fatalf("Expected route mount for '/items/[id]'")
	}
	if routeMount.ParamBinding != "id" && routeMount.ParamBinding != "[id]" {
		t.Errorf("Expected ParamBinding 'id' or '[id]', got '%s'", routeMount.ParamBinding)
	}

	// 3. Compile-time validation failure on invalid parameter
	_ = os.MkdirAll(filepath.Join(tempDir, "app", "items", "[unknown_param]"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "app", "items", "[unknown_param]", "page.html"), []byte("<h1>Invalid</h1>"), 0644)

	badPipeline, _ := compiler.NewPipeline(tempDir)
	_, badErr := badPipeline.Compile([]ir.TargetPlatform{ir.TargetWeb})
	if badErr == nil {
		t.Fatalf("Expected compile-time error for mismatched route parameter vs entity primary key, but got nil")
	}
}


