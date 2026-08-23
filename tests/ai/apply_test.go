package ai_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestAI_ApplyCodeSynthesis(t *testing.T) {
	app := harness.NewTestRunner(t, "ai-apply-test")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Run pogn ai "add customer feedback" --apply --dir=<path>
	projectDir := app.AppPath()
	out, err := app.RunCLI("ai", "add customer feedback", "--apply", "--dir="+projectDir)
	if err != nil {
		t.Fatalf("pogn ai --apply failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Applied Architectural Changes") {
		t.Errorf("Expected 'Applied Architectural Changes' in output, got:\n%s", out)
	}

	// Assert generated Go backend and HTML files exist
	app.AssertFileExists("server/feedback/entity.go")
	app.AssertFileExists("server/feedback/repository.go")
	app.AssertFileExists("server/feedback/service.go")
	app.AssertFileExists("server/feedback/handler.go")
	app.AssertFileExists("app/feedback/page.html")

	// Verify .ai architecture metadata was updated
	aiPath := filepath.Join(projectDir, ".pogonian", "architecture", "app.ai")
	data, err := os.ReadFile(aiPath)
	if err != nil {
		t.Fatalf("Failed to read app.ai: %v", err)
	}
	if !strings.Contains(string(data), "feedback") {
		t.Errorf("Expected 'feedback' module registered in app.ai")
	}

	// Verify project passes architecture checks with 0 violations!
	checkOut, err := app.RunCLI("check", projectDir)
	if err != nil {
		t.Fatalf("pogn check failed after AI code generation: %v\nOutput: %s", err, checkOut)
	}
	if !strings.Contains(checkOut, "All architecture and route checks passed") {
		t.Errorf("Expected architecture checks to pass cleanly, got:\n%s", checkOut)
	}
}
