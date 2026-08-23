package cli_test

import (
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestCLI_TreeCommand(t *testing.T) {
	app := harness.NewTestRunner(t, "tree-test-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Test pogn tree
	out, err := app.RunCLI("tree", projectDir)
	if err != nil {
		t.Fatalf("pogn tree failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Pogonian Application Structure Tree") {
		t.Errorf("Expected tree header in output:\n%s", out)
	}
	if !strings.Contains(out, "app/") {
		t.Errorf("Expected app/ in tree output:\n%s", out)
	}
	if !strings.Contains(out, "page.html") {
		t.Errorf("Expected page.html in tree output:\n%s", out)
	}

	// 2. Test pogn tree --json
	jsonOut, err := app.RunCLI("tree", "--json", projectDir)
	if err != nil {
		t.Fatalf("pogn tree --json failed: %v\nOutput: %s", err, jsonOut)
	}

	if !strings.Contains(jsonOut, `"name":`) || !strings.Contains(jsonOut, `"is_dir":`) {
		t.Errorf("Expected valid JSON output from pogn tree --json, got:\n%s", jsonOut)
	}
}
