package server_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/internal/server"
	"pogonian/tests/harness"
)

func TestServer_ServeApp(t *testing.T) {
	app := harness.NewTestRunner(t, "live-server-test")

	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	projectDir := app.AppPath()
	srv, err := server.New(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// 1. Test Home Page
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /, got %d", res.StatusCode)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "HTML-First. Powered by Go.") {
		t.Errorf("Expected homepage content, got:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "/_pogn/runtime.js") {
		t.Errorf("Expected runtime script injection, got:\n%s", bodyStr)
	}

	// 2. Test About Page
	res, err = http.Get(ts.URL + "/about")
	if err != nil {
		t.Fatalf("GET /about failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /about, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "About live-server-test") {
		t.Errorf("Expected about page content, got:\n%s", string(body))
	}

	// 3. Test Architecture Page
	res, err = http.Get(ts.URL + "/architecture")
	if err != nil {
		t.Fatalf("GET /architecture failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /architecture, got %d", res.StatusCode)
	}

	// 4. Test Static Asset (globals.css)
	res, err = http.Get(ts.URL + "/globals.css")
	if err != nil {
		t.Fatalf("GET /globals.css failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /globals.css, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "--bg-primary") {
		t.Errorf("Expected CSS tokens, got:\n%s", string(body))
	}

	// 5. Test Pogonian Client Runtime (/_pogn/runtime.js)
	res, err = http.Get(ts.URL + "/_pogn/runtime.js")
	if err != nil {
		t.Fatalf("GET /_pogn/runtime.js failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /_pogn/runtime.js, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "[Pogonian]") {
		t.Errorf("Expected runtime script, got:\n%s", string(body))
	}

	// 6. Test 404 Page
	res, err = http.Get(ts.URL + "/some-unknown-path")
	if err != nil {
		t.Fatalf("GET /some-unknown-path failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "Page Not Found") {
		t.Errorf("Expected 404 page content, got:\n%s", string(body))
	}

	fmt.Println("✓ All HTTP server endpoints and rendering verified successfully!")
}

func TestServer_ComponentTagAutoExpansionAndMultiPath(t *testing.T) {
	app := harness.NewTestRunner(t, "comp-expand-test")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Create a component in /components/module/footer.html
	rootModuleCompDir := filepath.Join(projectDir, "components", "module")
	_ = os.MkdirAll(rootModuleCompDir, 0755)
	_ = os.WriteFile(filepath.Join(rootModuleCompDir, "footer.html"), []byte(`<footer class="my-custom-footer"><p>&copy; 2026 Pogonian Framework</p></footer>`), 0644)

	// 2. Put <footer /> tag into app/about/page.html
	aboutPage := filepath.Join(projectDir, "app", "about", "page.html")
	_ = os.WriteFile(aboutPage, []byte(`<h1>About Us</h1><footer />`), 0644)

	srv, err := server.New(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/about")
	if err != nil {
		t.Fatalf("GET /about failed: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "my-custom-footer") {
		t.Errorf("Expected <footer /> to expand to footer component content from components/module/footer.html, got:\n%s", bodyStr)
	}
}

