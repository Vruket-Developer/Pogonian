package cli_test

import (
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestCLI_TestCommand(t *testing.T) {
	app := harness.NewTestRunner(t, "test-command-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Run pogn test (Architecture + Security)
	out, err := app.RunCLI("test", "--unit-only", projectDir)
	if err != nil {
		t.Fatalf("pogn test failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Architecture Contract Verification") || !strings.Contains(out, "Automated Security Scanner") {
		t.Errorf("Expected architecture and security verification phases in output:\n%s", out)
	}
}
