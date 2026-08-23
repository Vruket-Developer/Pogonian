package devtools_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pogonian/internal/devtools"
	"pogonian/internal/server"
	"pogonian/tests/harness"
)

func TestDevTools_DashboardAndDevBar(t *testing.T) {
	app := harness.NewTestRunner(t, "devtools-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	srv, err := server.New(app.AppPath(), true)
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}
	defer srv.Close()

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// 1. Record telemetry request
	devtools.GlobalRecorder.Record("GET", "/products", http.StatusOK, 500*time.Microsecond)

	// 2. Test GET /__pogn/devtools
	res, err := http.Get(ts.URL + "/__pogn/devtools")
	if err != nil {
		t.Fatalf("GET /__pogn/devtools failed: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for devtools, got %d", res.StatusCode)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Pogonian DevTools") {
		t.Errorf("Expected DevTools title, got:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Live Architecture Dependency Graph") {
		t.Errorf("Expected Architecture graph section in DevTools")
	}

	// 3. Test GET /__pogn/devbar.js
	res, err = http.Get(ts.URL + "/__pogn/devbar.js")
	if err != nil {
		t.Fatalf("GET /__pogn/devbar.js failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for devbar.js, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "pogn-devbar") {
		t.Errorf("Expected pogn-devbar script content")
	}

	// 4. Test DevBar injected into normal page HTML
	res, err = http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()

	if !strings.Contains(string(body), "/__pogn/devbar.js") {
		t.Errorf("Expected DevBar script tag injected in dev mode")
	}
}
