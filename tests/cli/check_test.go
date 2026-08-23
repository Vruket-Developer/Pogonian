package cli_test

import (
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestPognCheck_ValidProject(t *testing.T) {
	app := harness.NewTestRunner(t, "check-test-app")

	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	out, err := app.RunCLI("check", app.AppPath())
	if err != nil {
		t.Fatalf("pogn check failed on valid app: %v\nOutput:\n%s", err, out)
	}

	if !strings.Contains(out, "All architecture and route checks passed") {
		t.Errorf("Expected success confirmation, got:\n%s", out)
	}
}
