package cli_test

import (
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestCLI_DoctorDiagnostics(t *testing.T) {
	app := harness.NewTestRunner(t, "doctor-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()
	out, err := app.RunCLI("doctor", projectDir)
	if err != nil {
		t.Fatalf("pogn doctor failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Pogonian Environment & Project Doctor") {
		t.Errorf("Expected doctor header, got:\n%s", out)
	}
	if !strings.Contains(out, "Go Toolchain") {
		t.Errorf("Expected Go Toolchain check in doctor output")
	}
}

func TestCLI_ExplainAndBlueprint(t *testing.T) {
	app := harness.NewTestRunner(t, "explain-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Test pogn explain
	explainOut, err := app.RunCLI("explain", "--dir="+projectDir)
	if err != nil {
		t.Fatalf("pogn explain failed: %v\nOutput: %s", err, explainOut)
	}
	if !strings.Contains(explainOut, "Architecture Explanation") {
		t.Errorf("Expected explain header, got:\n%s", explainOut)
	}

	// 2. Test pogn blueprint
	bpOut, err := app.RunCLI("blueprint", projectDir)
	if err != nil {
		t.Fatalf("pogn blueprint failed: %v\nOutput: %s", err, bpOut)
	}
	if !strings.Contains(bpOut, "Architecture Blueprint") || !strings.Contains(bpOut, "mermaid") {
		t.Errorf("Expected mermaid blueprint, got:\n%s", bpOut)
	}

	// 3. Test pogn docs
	docsOut, err := app.RunCLI("docs", projectDir)
	if err != nil {
		t.Fatalf("pogn docs failed: %v\nOutput: %s", err, docsOut)
	}
	if !strings.Contains(docsOut, "API Documentation") {
		t.Errorf("Expected API docs header, got:\n%s", docsOut)
	}
}

func TestCLI_CleanAndReset(t *testing.T) {
	app := harness.NewTestRunner(t, "clean-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Test pogn clean
	cleanOut, err := app.RunCLI("clean", projectDir)
	if err != nil {
		t.Fatalf("pogn clean failed: %v\nOutput: %s", err, cleanOut)
	}
	if !strings.Contains(cleanOut, "Cleaning Pogonian Project") {
		t.Errorf("Expected clean message, got:\n%s", cleanOut)
	}

	// 2. Test pogn reset
	resetOut, err := app.RunCLI("reset", projectDir)
	if err != nil {
		t.Fatalf("pogn reset failed: %v\nOutput: %s", err, resetOut)
	}
	if !strings.Contains(resetOut, "reset to fresh state") {
		t.Errorf("Expected reset message, got:\n%s", resetOut)
	}
}
