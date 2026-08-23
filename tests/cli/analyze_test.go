package cli_test

import (
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestPognAnalyze_Formats(t *testing.T) {
	app := harness.NewTestRunner(t, "analyze-cli-test")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// 1. Test ASCII output
	out, err := app.RunCLI("analyze", app.AppPath())
	if err != nil {
		t.Fatalf("pogn analyze failed: %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "Architecture Graph") {
		t.Errorf("Expected Architecture Graph header in ASCII format, got:\n%s", out)
	}

	// 2. Test Mermaid format
	out, err = app.RunCLI("analyze", "--format=mermaid", app.AppPath())
	if err != nil {
		t.Fatalf("pogn analyze --format=mermaid failed: %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "flowchart TD") {
		t.Errorf("Expected flowchart TD in mermaid format, got:\n%s", out)
	}

	// 3. Test JSON format
	out, err = app.RunCLI("analyze", "--format=json", app.AppPath())
	if err != nil {
		t.Fatalf("pogn analyze --format=json failed: %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, `"framework": "pogonian"`) {
		t.Errorf("Expected JSON IR output, got:\n%s", out)
	}

	// 4. Test Inspect command
	out, err = app.RunCLI("inspect", app.AppPath())
	if err != nil {
		t.Fatalf("pogn inspect failed: %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "Pogonian Architecture Inspector") {
		t.Errorf("Expected inspector output, got:\n%s", out)
	}
}
