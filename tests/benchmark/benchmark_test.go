package benchmark_test

import (
	"strings"
	"testing"

	"pogonian/internal/benchmark"
	"pogonian/tests/harness"
)

func TestBenchmark_Evaluation(t *testing.T) {
	app := harness.NewTestRunner(t, "benchmark-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	eval, err := benchmark.RunEvaluation(app.AppPath())
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if len(eval.Metrics) < 4 {
		t.Errorf("Expected at least 4 research metrics, got %d", len(eval.Metrics))
	}

	summary := eval.Summary()
	if !strings.Contains(summary, "MSc Empirical Research Evaluation") {
		t.Errorf("Expected MSc evaluation header in summary, got:\n%s", summary)
	}
	if !strings.Contains(summary, "AI Context Tokens") {
		t.Errorf("Expected AI Context Tokens metric in summary")
	}
}
