package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/internal/openapi"
	"pogonian/internal/server"
	"pogonian/tests/harness"
)

func TestSwaggerAndAPIEngine(t *testing.T) {
	app := harness.NewTestRunner(t, "swagger-api-test")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	projectDir := app.AppPath()
	srv, err := server.New(projectDir, true) // DevMode = true
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// 1. Test Swagger UI Page (/swagger)
	res, err := http.Get(ts.URL + "/swagger")
	if err != nil {
		t.Fatalf("GET /swagger failed: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /swagger, got %d", res.StatusCode)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "SwaggerUIBundle") || !strings.Contains(bodyStr, "swagger-ui") {
		t.Errorf("Expected Swagger UI HTML page, got:\n%s", bodyStr)
	}

	// 2. Test OpenAPI 3.0 JSON Spec Endpoint (/swagger.json)
	res, err = http.Get(ts.URL + "/swagger.json")
	if err != nil {
		t.Fatalf("GET /swagger.json failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /swagger.json, got %d", res.StatusCode)
	}

	var spec openapi.Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		t.Fatalf("Failed to parse OpenAPI JSON spec: %v\nBody:\n%s", err, string(body))
	}

	if spec.OpenAPI != "3.0.0" {
		t.Errorf("Expected OpenAPI version 3.0.0, got %s", spec.OpenAPI)
	}

	// Verify paths discovered in OpenAPI spec
	expectedPaths := []string{"/", "/about", "/architecture", "/products", "/login", "/register", "/logout", "/api/health"}
	for _, p := range expectedPaths {
		if _, exists := spec.Paths[p]; !exists {
			t.Errorf("Expected OpenAPI path %s to be present in spec", p)
		}
	}

	// 3. Test API Health Check Endpoint (/api/health)
	res, err = http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /api/health, got %d", res.StatusCode)
	}

	var healthResp map[string]interface{}
	if err := json.Unmarshal(body, &healthResp); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}
	if healthResp["status"] != "ok" || healthResp["framework"] != "pogonian" {
		t.Errorf("Unexpected health check response: %v", healthResp)
	}

	// 4. Test API Routes Listing (/api/routes)
	res, err = http.Get(ts.URL + "/api/routes")
	if err != nil {
		t.Fatalf("GET /api/routes failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /api/routes, got %d", res.StatusCode)
	}

	var routesResp map[string]interface{}
	if err := json.Unmarshal(body, &routesResp); err != nil {
		t.Fatalf("Failed to parse /api/routes: %v", err)
	}
	totalRoutes, _ := routesResp["total"].(float64)
	if totalRoutes <= 0 {
		t.Errorf("Expected positive route count, got %v", totalRoutes)
	}

	// 5. Test Dynamic Zero-Restart Route Rescanning in Dev Mode
	newRouteDir := filepath.Join(projectDir, "app", "dynroute")
	_ = os.MkdirAll(newRouteDir, 0755)
	_ = os.WriteFile(filepath.Join(newRouteDir, "page.html"), []byte(`<h1>Dynamic Route Without Restart</h1>`), 0644)

	res, err = http.Get(ts.URL + "/dynroute")
	if err != nil {
		t.Fatalf("GET /dynroute failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for new dynamic route without restart, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "Dynamic Route Without Restart") {
		t.Errorf("Expected dynamic route content, got:\n%s", string(body))
	}

	// 6. Test Dynamic Zero-Restart Component Discovery in Dev Mode
	newCompDir := filepath.Join(projectDir, "app", "components")
	_ = os.MkdirAll(newCompDir, 0755)
	_ = os.WriteFile(filepath.Join(newCompDir, "badge.html"), []byte(`<span class="badge">Dynamic Badge</span>`), 0644)

	// Update page to use <badge />
	_ = os.WriteFile(filepath.Join(newRouteDir, "page.html"), []byte(`<h1>Page With Badge</h1><badge />`), 0644)

	res, err = http.Get(ts.URL + "/dynroute")
	if err != nil {
		t.Fatalf("GET /dynroute with component failed: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "Dynamic Badge") {
		t.Errorf("Expected new component to be compiled without restart, got:\n%s", string(body))
	}
}
