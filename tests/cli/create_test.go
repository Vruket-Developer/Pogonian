package cli_test

import (
	"os"
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestPognCreate_BasicProject(t *testing.T) {
	app := harness.NewTestRunner(t, "my-super-app")

	// 1. Create app
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// 2. Assert all critical files exist
	app.AssertFileExists("pogonian.yaml")
	app.AssertFileExists("pogonian.lock")
	app.AssertFileExists("package.json")
	app.AssertFileExists("globals.css")
	app.AssertFileExists("globals.js")
	app.AssertFileExists(".gitignore")

	// App & routes
	app.AssertFileExists("app/layout.html")
	app.AssertFileExists("app/page.html")
	app.AssertFileExists("app/about/page.html")
	app.AssertFileExists("app/architecture/page.html")

	// Components
	app.AssertFileExists("app/components/Card.html")

	// Backend server starter
	app.AssertFileExists("server/routes.go")

	// Documentation & Readme
	app.AssertFileExists("README.md")
	app.AssertFileExists("docs/README.md")

	// Architecture & AI metadata
	app.AssertFileExists(".pogonian/architecture/app.ai")

	// 3. Assert template interpolation works
	app.AssertFileContains("pogonian.yaml", "name: my-super-app")
	app.AssertFileContains(".pogonian/architecture/app.ai", "application: my-super-app")
	app.AssertFileContains("package.json", `"name": "my-super-app"`)
	app.AssertFileContains("app/about/page.html", "my-super-app")
}

func TestPognCreate_DryRun(t *testing.T) {
	app := harness.NewTestRunner(t, "dry-run-test")

	out, err := app.RunCLI("create", "dry-run-test", "--dry-run", app.AppPath())
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "[Dry Run]") {
		t.Errorf("Expected dry run message, got: %s", out)
	}

	// File should NOT exist
	targetYaml := app.AppPath("pogonian.yaml")
	if _, err := os.Stat(targetYaml); err == nil {
		t.Errorf("Expected pogonian.yaml NOT to exist after dry run")
	}
}

func TestPognCreate_InvalidName(t *testing.T) {
	app := harness.NewTestRunner(t, "invalid_runner")

	out, err := app.RunCLI("create", "invalid name with spaces!")
	if err == nil {
		t.Fatalf("Expected error for invalid project name, but succeeded.\nOutput: %s", out)
	}

	if !strings.Contains(out, "invalid project name") {
		t.Errorf("Expected 'invalid project name' error, got: %s", out)
	}
}

func osStat(path string) (any, error) {
	return harness.NewTestRunner(&testing.T{}, "temp").AppPath(path), nil
}
