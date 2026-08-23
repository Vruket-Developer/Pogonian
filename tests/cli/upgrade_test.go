package cli_test

import (
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestCLI_UpgradeCheckAndCompatibility(t *testing.T) {
	app := harness.NewTestRunner(t, "upgrade-check-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Test pogn version -list
	out, err := app.RunCLI("version", "-list", projectDir)
	if err != nil {
		t.Fatalf("pogn version -list failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Pogonian Version History & Compatibility Matrix") {
		t.Errorf("Expected version history header in output:\n%s", out)
	}
	if !strings.Contains(out, "max upgradable") {
		t.Errorf("Expected '(max upgradable)' in output:\n%s", out)
	}
	if !strings.Contains(out, "0.2.0") {
		t.Errorf("Expected version 0.2.0 listed in available versions")
	}

	// 2. Test pogn upgrade --dry-run
	dryOut, err := app.RunCLI("upgrade", "--dry-run", projectDir)
	if err != nil {
		t.Fatalf("pogn upgrade --dry-run failed: %v\nOutput: %s", err, dryOut)
	}

	if !strings.Contains(dryOut, "[Dry Run Mode]") {
		t.Errorf("Expected dry run preview output, got:\n%s", dryOut)
	}
}
