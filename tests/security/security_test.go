package security_test

import (
	"os"
	"path/filepath"
	"testing"

	"pogonian/internal/security"
	"pogonian/tests/harness"
)

func TestSecurity_AuditCleanProject(t *testing.T) {
	app := harness.NewTestRunner(t, "security-clean-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	report, err := security.AuditProject(app.AppPath())
	if err != nil {
		t.Fatalf("Audit failed: %v", err)
	}

	if report.Score < 90 {
		t.Errorf("Expected high security score on default project, got %d", report.Score)
	}
}

func TestSecurity_DetectVulnerabilities(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Create file with fake exposed AWS key
	secretFile := filepath.Join(tempDir, "config.go")
	_ = os.WriteFile(secretFile, []byte(`package main
const awsKey = "AKIAIOSFODNN7EXAMPLE" // Exposed Key!
`), 0644)

	// 2. Create file with SQL injection concatenation
	sqlFile := filepath.Join(tempDir, "repo.go")
	_ = os.WriteFile(sqlFile, []byte(`package main
import "fmt"
func getUser(id string) {
	query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", id)
	_ = query
}
`), 0644)

	report, err := security.AuditProject(tempDir)
	if err != nil {
		t.Fatalf("Audit failed: %v", err)
	}

	if len(report.Findings) < 2 {
		t.Fatalf("Expected at least 2 security findings, got %d", len(report.Findings))
	}

	foundSecret := false
	foundSQLi := false
	for _, f := range report.Findings {
		if f.Type == "SECRET_EXPOSURE" {
			foundSecret = true
		}
		if f.Type == "SQL_INJECTION" {
			foundSQLi = true
		}
	}

	if !foundSecret {
		t.Errorf("Expected secret exposure finding")
	}
	if !foundSQLi {
		t.Errorf("Expected SQL injection finding")
	}
}
