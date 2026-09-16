package e2e_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/pkg/compiler"
	"pogonian/pkg/ir"
	"pogonian/pkg/server"
	"pogonian/tests/harness"
)

func TestE2E_FullLifecycleAndMultiTarget(t *testing.T) {
	app := harness.NewTestRunner(t, "e2e-universal-app")

	// =========================================================================
	// Step 1: Create Project
	// =========================================================================
	t.Log("Step 1: Scaffolding fresh project...")
	if err := app.CreateApp(); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}
	projectDir := app.AppPath()

	// Verify project layout
	app.AssertFileExists("pogonian.yaml")
	app.AssertFileExists("app/page.html")
	app.AssertFileExists("app/layout.html")
	app.AssertFileExists(".pogonian/security/rules.yaml")

	// =========================================================================
	// Step 2: Route Discovery & UI Rendering
	// =========================================================================
	t.Log("Step 2: Starting server and verifying route discovery & rendering...")
	srv, err := server.New(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// GET /
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for /, got %d. Body: %s", res.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "runtime.js") {
		t.Errorf("Expected runtime.js injected in HTML")
	}

	// GET /about
	resAbout, err := http.Get(ts.URL + "/about")
	if err != nil {
		t.Fatalf("GET /about failed: %v", err)
	}
	resAbout.Body.Close()
	if resAbout.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for /about, got %d", resAbout.StatusCode)
	}

	// =========================================================================
	// Step 3: Backend & Database Connection (Mount Flow)
	// =========================================================================
	t.Log("Step 3: Verifying backend action and data persistence...")
	getProdRes, err := http.Get(ts.URL + "/products")
	if err != nil {
		t.Fatalf("GET /products failed: %v", err)
	}
	getProdRes.Body.Close()

	csrfCookie := ""
	for _, c := range getProdRes.Cookies() {
		if c.Name == "_pogn_csrf" {
			csrfCookie = c.Value
		}
	}

	formVals := url.Values{
		"name":  {"Ergonomic Mechanical Keyboard"},
		"price": {"149.99"},
		"_csrf": {csrfCookie},
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // capture 303 redirect
		},
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/products", strings.NewReader(formVals.Encode()))
	if err != nil {
		t.Fatalf("Failed to create POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if csrfCookie != "" {
		req.AddCookie(&http.Cookie{Name: "_pogn_csrf", Value: csrfCookie})
	}
	postRes, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /products failed: %v", err)
	}
	postRes.Body.Close()
	if postRes.StatusCode != http.StatusSeeOther {
		t.Fatalf("Expected 303 redirect from action, got %d", postRes.StatusCode)
	}

	// Fetch dynamic items via API
	getRes, err := http.Get(ts.URL + "/api/products")
	if err != nil {
		t.Fatalf("GET /api/products failed: %v", err)
	}
	apiBody, _ := io.ReadAll(getRes.Body)
	getRes.Body.Close()

	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for /api/products, got %d", getRes.StatusCode)
	}
	if !strings.Contains(string(apiBody), "Ergonomic Mechanical Keyboard") {
		t.Errorf("Expected product in JSON API response, got: %s", string(apiBody))
	}

	// =========================================================================
	// Step 4: Multi-Target Native Compilation (All Platforms)
	// =========================================================================
	t.Log("Step 4: Compiling universal application to all 7 native targets...")
	pipeline, err := compiler.NewPipeline(projectDir)
	if err != nil {
		t.Fatalf("Failed to initialize compiler pipeline: %v", err)
	}

	targets := ir.AllTargetPlatforms()
	compResult, err := pipeline.Compile(targets)
	if err != nil {
		t.Fatalf("Universal compilation failed: %v", err)
	}

	// Verify every target platform was successfully lowered
	for _, target := range targets {
		art, exists := compResult.Artifacts[target]
		if !exists {
			t.Fatalf("Target %s was not lowered in compilation result", target)
		}
		if len(art.Files) == 0 {
			t.Errorf("Target %s produced 0 files", target)
		}
		t.Logf("  ✓ Successfully lowered [%s]: %d files generated (%s)", target, len(art.Files), art.EntrypointFile)
	}

	// 4a. Verify Android Output
	androidArt := compResult.Artifacts[ir.TargetAndroid]
	var hasRootView bool
	for path := range androidArt.Files {
		if strings.HasSuffix(path, "PogonianRootView.kt") {
			hasRootView = true
			break
		}
	}
	if len(androidArt.Files["android/build.gradle.kts"]) == 0 ||
		len(androidArt.Files["android/app/src/main/AndroidManifest.xml"]) == 0 ||
		!hasRootView {
		t.Errorf("Android target missing essential build files")
	}

	// 4b. Verify iOS Output
	iosArt := compResult.Artifacts[ir.TargetIOS]
	if len(iosArt.Files["ios/Package.swift"]) == 0 ||
		len(iosArt.Files["ios/Info.plist"]) == 0 ||
		len(iosArt.Files["ios/Sources/PogonianRootView.swift"]) == 0 {
		t.Errorf("iOS target missing essential Swift Package files")
	}

	// 4c. Verify Desktop Outputs (Linux, Windows, macOS)
	for _, desk := range []ir.TargetPlatform{ir.TargetLinux, ir.TargetWindows, ir.TargetMacOS} {
		deskArt := compResult.Artifacts[desk]
		goModFile := fmt.Sprintf("desktop/%s/go.mod", desk)
		mainFile := fmt.Sprintf("desktop/%s/main.go", desk)
		if len(deskArt.Files[goModFile]) == 0 || len(deskArt.Files[mainFile]) == 0 {
			t.Errorf("Desktop target %s missing go.mod or main.go", desk)
		}
	}

	// 4d. Verify Android TV Output
	tvArt := compResult.Artifacts[ir.TargetAndroidTV]
	if len(tvArt.Files["android_tv/src/main/AndroidManifest.xml"]) == 0 ||
		len(tvArt.Files["android_tv/src/main/kotlin/com/pogonian/tv/TVActivity.kt"]) == 0 {
		t.Errorf("Android TV target missing Leanback activity files")
	}

	// 4e. Build Validation: Compile Native Desktop Target Binary with go build
	t.Log("  🔨 Step 4e: Validating genuine native desktop compilation via Go toolchain...")
	deskDir := filepath.Join(projectDir, "desktop_build_test")
	deskArt := compResult.Artifacts[ir.TargetLinux]
	for relPath, content := range deskArt.Files {
		cleanRel := strings.TrimPrefix(relPath, "desktop/linux/")
		fPath := filepath.Join(deskDir, cleanRel)
		if err := os.MkdirAll(filepath.Dir(fPath), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(fPath, content, 0644); err != nil {
			t.Fatalf("Failed to write desktop source file: %v", err)
		}
	}
	cmd := exec.Command("go", "build", "-o", "desktop-app", ".")
	cmd.Dir = deskDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile lowered native desktop application: %v\nOutput: %s", err, string(out))
	}
	if _, err := os.Stat(filepath.Join(deskDir, "desktop-app")); err != nil {
		t.Fatalf("Expected compiled desktop binary, got error: %v", err)
	}
	t.Log("  ✓ Successfully compiled native desktop application to working binary!")

	// =========================================================================
	// Step 5: Standalone Production Release Build
	// =========================================================================
	t.Log("Step 5: Executing standalone production build...")
	buildOut, err := app.RunCLI("build", projectDir)
	if err != nil {
		t.Fatalf("pogn build failed: %v\nOutput: %s", err, buildOut)
	}

	binPath := filepath.Join(projectDir, "dist", app.Name)
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("Expected production binary at %s, got error: %v", binPath, err)
	}

	t.Logf("✨ Complete end-to-end application lifecycle verified successfully for %s!", app.Name)
}
