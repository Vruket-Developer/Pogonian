package modules_test

import (
	"os"
	"path/filepath"
	"testing"

	"pogonian/internal/modules"
)

func TestModules_InstallHeaderComponent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogn_mod_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create fake app structure
	_ = os.MkdirAll(filepath.Join(tempDir, "app"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "globals.css"), []byte("/* Base Styles */\n"), 0644)

	// Install header component
	res, err := modules.InstallModule(tempDir, "header")
	if err != nil {
		t.Fatalf("InstallModule header failed: %v", err)
	}

	if res.ModuleName != "header" || res.Category != modules.CategoryUIComponent {
		t.Errorf("Unexpected module result: %v", res)
	}

	headerFile := filepath.Join(tempDir, "app", "components", "header.html")
	if _, err := os.Stat(headerFile); err != nil {
		t.Errorf("Expected header.html to exist at %s", headerFile)
	}
}

func TestModules_InstallFooterComponent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogn_footer_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create fake app structure
	_ = os.MkdirAll(filepath.Join(tempDir, "app"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "globals.css"), []byte("/* Base Styles */\n"), 0644)

	// Install footer component
	res, err := modules.InstallModule(tempDir, "footer")
	if err != nil {
		t.Fatalf("InstallModule footer failed: %v", err)
	}

	if res.ModuleName != "footer" || res.Category != modules.CategoryUIComponent {
		t.Errorf("Unexpected module result: %v", res)
	}

	footerFile := filepath.Join(tempDir, "app", "components", "footer.html")
	if _, err := os.Stat(footerFile); err != nil {
		t.Errorf("Expected footer.html to exist at %s", footerFile)
	}
}

func TestModules_InstallLoginModule(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogn_login_mod_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Install login module
	res, err := modules.InstallModule(tempDir, "login")
	if err != nil {
		t.Fatalf("InstallModule login failed: %v", err)
	}

	if res.ModuleName != "login" || res.Category != modules.CategoryFullStack {
		t.Errorf("Unexpected login module result: %v", res)
	}

	loginPage := filepath.Join(tempDir, "app", "login", "page.html")
	if _, err := os.Stat(loginPage); err != nil {
		t.Errorf("Expected app/login/page.html to exist")
	}
}
