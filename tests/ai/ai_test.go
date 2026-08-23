package ai_test

import (
	"strings"
	"testing"

	"pogonian/internal/ai"
	"pogonian/internal/architecture"
	"pogonian/tests/harness"
)

func TestAI_ContextAndPlanner(t *testing.T) {
	app := harness.NewTestRunner(t, "ai-planner-test")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Test .ai parser
	spec, err := ai.ParseAISpec(projectDir)
	if err != nil {
		t.Fatalf("Failed to parse app.ai: %v", err)
	}
	if spec.Application != "ai-planner-test" {
		t.Errorf("Expected app name 'ai-planner-test', got '%s'", spec.Application)
	}

	// 2. Test Architecture Context Extraction
	analyzer, err := architecture.NewAnalyzer(projectDir)
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}
	appIR, graph, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	engine := ai.NewContextEngine(projectDir, appIR, graph)
	ctx, err := engine.ExtractContext("Add product discount coupons")
	if err != nil {
		t.Fatalf("Failed to extract context: %v", err)
	}

	if ctx.SavingsPercent <= 0 {
		t.Errorf("Expected token savings > 0%%, got %.2f%%", ctx.SavingsPercent)
	}
	if len(ctx.RelevantFiles) == 0 {
		t.Errorf("Expected targeted relevant files, got 0")
	}

	// 3. Test Architectural Plan Generation
	plan := ai.GeneratePlan("Add product reviews", ctx)
	if len(plan.Steps) != 7 {
		t.Errorf("Expected 7 architectural layer steps, got %d", len(plan.Steps))
	}
	if !strings.Contains(plan.Explanation, "Page → Handler → Service → Repository") {
		t.Errorf("Expected Pogonian flow in explanation")
	}
}
