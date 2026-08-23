package ai_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/internal/ai"
	"pogonian/tests/harness"
)

func TestAI_SandboxAndRollback(t *testing.T) {
	app := harness.NewTestRunner(t, "ai-sandbox-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()
	sandbox := ai.NewSandboxEngine(projectDir)

	// 1. Create Snapshot before modifications
	snap, err := sandbox.CreateSnapshot("Test adding customers feature")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	if snap == nil || snap.ID == "" {
		t.Fatalf("Expected valid snapshot instance")
	}

	// 2. Modify a file
	layoutPath := filepath.Join(projectDir, "app", "layout.html")
	origBytes, _ := os.ReadFile(layoutPath)
	_ = os.WriteFile(layoutPath, []byte("MODIFIED_CONTENT_FOR_TEST"), 0644)

	// 3. Perform Rollback
	restored, err := sandbox.Rollback(snap.ID)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if restored == nil {
		t.Fatalf("Expected restored snapshot info")
	}

	// 4. Verify original content restored
	afterRollback, _ := os.ReadFile(layoutPath)
	if string(afterRollback) != string(origBytes) {
		t.Errorf("Rollback failed to restore original file content")
	}
	if strings.Contains(string(afterRollback), "MODIFIED_CONTENT_FOR_TEST") {
		t.Errorf("Modified content still present after rollback")
	}
}
