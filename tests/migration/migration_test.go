package migration_test

import (
	"os"
	"path/filepath"
	"testing"

	"pogonian/internal/migration"
)

func TestMigration_ReactNextProject(t *testing.T) {
	legacyDir := t.TempDir()

	// 1. Setup mock Next.js legacy project
	pkgJSON := `{
		"name": "legacy-next-store",
		"version": "1.0.0",
		"dependencies": {
			"next": "14.0.0",
			"react": "18.2.0"
		}
	}`
	_ = os.WriteFile(filepath.Join(legacyDir, "package.json"), []byte(pkgJSON), 0644)

	_ = os.MkdirAll(filepath.Join(legacyDir, "app", "dashboard"), 0755)
	_ = os.MkdirAll(filepath.Join(legacyDir, "app", "settings"), 0755)
	_ = os.MkdirAll(filepath.Join(legacyDir, "components"), 0755)

	_ = os.WriteFile(filepath.Join(legacyDir, "app", "page.tsx"), []byte(`export default function Home() { return <h1>Home</h1>; }`), 0644)
	_ = os.WriteFile(filepath.Join(legacyDir, "app", "dashboard", "page.tsx"), []byte(`export default function Dashboard() { return <h1>Dashboard</h1>; }`), 0644)
	_ = os.WriteFile(filepath.Join(legacyDir, "app", "settings", "page.tsx"), []byte(`export default function Settings() { return <h1>Settings</h1>; }`), 0644)
	_ = os.WriteFile(filepath.Join(legacyDir, "components", "Navbar.tsx"), []byte(`export function Navbar() { return <nav></nav>; }`), 0644)

	// 2. Run Migration Analysis
	appIR, report, err := migration.MigrateReactProject(legacyDir)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	if report.SourceFramework != "Next.js" {
		t.Errorf("Expected framework 'Next.js', got '%s'", report.SourceFramework)
	}
	if report.RoutesCount < 3 {
		t.Errorf("Expected at least 3 routes, got %d", report.RoutesCount)
	}
	if report.ComponentsCount < 1 {
		t.Errorf("Expected at least 1 component, got %d", report.ComponentsCount)
	}

	// 3. Generate Pogonian project from IR
	targetDir := filepath.Join(t.TempDir(), "migrated-app")
	err = migration.GenerateFromIR(appIR, targetDir)
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	// Verify migrated Pogonian files exist
	if _, err := os.Stat(filepath.Join(targetDir, "pogonian.yaml")); err != nil {
		t.Errorf("pogonian.yaml missing in migrated app")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "app", "page.html")); err != nil {
		t.Errorf("app/page.html missing in migrated app")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "app", "dashboard", "page.html")); err != nil {
		t.Errorf("app/dashboard/page.html missing in migrated app")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "app", "settings", "page.html")); err != nil {
		t.Errorf("app/settings/page.html missing in migrated app")
	}
}
