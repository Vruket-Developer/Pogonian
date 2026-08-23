package architecture_test

import (
	"strings"
	"testing"

	"pogonian/internal/architecture"
	"pogonian/tests/harness"
)

func TestArchitecture_ChangeImpactAnalysis(t *testing.T) {
	app := harness.NewTestRunner(t, "impact-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Analyze Service Impact
	report, err := architecture.AnalyzeImpact(projectDir, "server/products/service.go")
	if err != nil {
		t.Fatalf("AnalyzeImpact failed: %v", err)
	}

	if report.TargetType != "service" {
		t.Errorf("Expected target type 'service', got '%s'", report.TargetType)
	}

	if len(report.AffectedRoutes) == 0 {
		t.Errorf("Expected affected routes for products service")
	}

	if report.ImpactLevel == "" {
		t.Errorf("Expected non-empty impact level")
	}

	// 2. Test CLI command pogn impact products
	out, err := app.RunCLI("impact", "products", "--dir="+projectDir)
	if err != nil {
		t.Fatalf("pogn impact products failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Change Impact Analysis") {
		t.Errorf("Expected Change Impact Analysis header, got:\n%s", out)
	}
	if !strings.Contains(out, "Routes") {
		t.Errorf("Expected Routes section in impact output")
	}
}
