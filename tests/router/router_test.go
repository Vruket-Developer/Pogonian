package router_test

import (
	"os"
	"path/filepath"
	"testing"

	"pogonian/pkg/router"
)

func TestRouter_FilesystemDiscovery(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "app")

	// Create structure:
	// app/layout.html
	// app/page.html
	// app/about/page.html
	// app/products/[id]/page.html
	// app/products/layout.html
	_ = os.MkdirAll(filepath.Join(appDir, "about"), 0755)
	_ = os.MkdirAll(filepath.Join(appDir, "products", "[id]"), 0755)

	_ = os.WriteFile(filepath.Join(appDir, "layout.html"), []byte("Root Layout {{.Content}}"), 0644)
	_ = os.WriteFile(filepath.Join(appDir, "page.html"), []byte("Home Page"), 0644)
	_ = os.WriteFile(filepath.Join(appDir, "about", "page.html"), []byte("About Page"), 0644)
	_ = os.WriteFile(filepath.Join(appDir, "products", "layout.html"), []byte("Product Layout {{.Content}}"), 0644)
	_ = os.WriteFile(filepath.Join(appDir, "products", "[id]", "page.html"), []byte("Product Detail"), 0644)

	r, err := router.New(appDir)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	if len(r.Routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d", len(r.Routes))
	}

	// Test 1: Match root
	rt, params := r.Match("/")
	if rt == nil {
		t.Fatalf("Expected match for '/', got nil")
	}
	if rt.Path != "/" {
		t.Errorf("Expected route path '/', got '%s'", rt.Path)
	}
	if len(rt.LayoutPaths) != 1 {
		t.Errorf("Expected 1 root layout, got %d", len(rt.LayoutPaths))
	}

	// Test 2: Match about
	rt, params = r.Match("/about")
	if rt == nil {
		t.Fatalf("Expected match for '/about', got nil")
	}
	if rt.Path != "/about" {
		t.Errorf("Expected route path '/about', got '%s'", rt.Path)
	}

	// Test 3: Match parameterized product route
	rt, params = r.Match("/products/42")
	if rt == nil {
		t.Fatalf("Expected match for '/products/42', got nil")
	}
	if rt.Path != "/products/:id" {
		t.Errorf("Expected route path '/products/:id', got '%s'", rt.Path)
	}
	if params["id"] != "42" {
		t.Errorf("Expected param id='42', got '%s'", params["id"])
	}
	if len(rt.LayoutPaths) != 2 {
		t.Errorf("Expected 2 nested layouts (root + product), got %d", len(rt.LayoutPaths))
	}

	// Test 4: 404 non-existent
	rt, _ = r.Match("/non-existent")
	if rt != nil {
		t.Errorf("Expected nil match for '/non-existent', got %v", rt)
	}
}
