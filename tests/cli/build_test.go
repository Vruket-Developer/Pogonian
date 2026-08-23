package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestCLI_BuildStandalone(t *testing.T) {
	app := harness.NewTestRunner(t, "build-standalone-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()
	out, err := app.RunCLI("build", "--standalone", projectDir)
	if err != nil {
		t.Fatalf("pogn build --standalone failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Standalone production build complete") {
		t.Errorf("Expected build success summary, got:\n%s", out)
	}

	distBin := filepath.Join(projectDir, "dist", "build-standalone-app")
	if _, err := os.Stat(distBin); err != nil {
		t.Errorf("Expected compiled binary at %s", distBin)
	}
}
