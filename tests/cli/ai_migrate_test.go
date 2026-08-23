package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestPognAI_CLI(t *testing.T) {
	app := harness.NewTestRunner(t, "ai-cli-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	out, err := app.RunCLI("ai", "Add wishlist functionality")
	if err != nil {
		t.Fatalf("pogn ai failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Pogonian AI Architecture Planner") {
		t.Errorf("Expected AI planner output, got:\n%s", out)
	}
	if !strings.Contains(out, "tokens saved") {
		t.Errorf("Expected token savings line, got:\n%s", out)
	}
}

func TestPognImport_CLI(t *testing.T) {
	app := harness.NewTestRunner(t, "import-cli-app")

	legacyDir := filepath.Join(t.TempDir(), "fake-react")
	_ = os.MkdirAll(legacyDir, 0755)
	_ = os.WriteFile(filepath.Join(legacyDir, "package.json"), []byte(`{"name":"fake-react","dependencies":{"react":"18.0.0"}}`), 0644)

	out, err := app.RunCLI("import", legacyDir)
	if err != nil {
		t.Fatalf("pogn import failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Pogonian Migration Report") {
		t.Errorf("Expected migration report in output, got:\n%s", out)
	}
	if !strings.Contains(out, "React") {
		t.Errorf("Expected React detected, got:\n%s", out)
	}
}
