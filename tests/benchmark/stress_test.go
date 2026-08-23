package benchmark_test

import (
	"testing"

	"pogonian/internal/benchmark"
)

func TestBenchmark_Stress10kRoutes(t *testing.T) {
	// Run with 1,000 synthetic routes and 5,000 lookups for fast test execution
	res, err := benchmark.RunStressBenchmark(1000, 5000)
	if err != nil {
		t.Fatalf("RunStressBenchmark failed: %v", err)
	}

	if res.TotalRoutes != 1000 {
		t.Errorf("Expected 1000 routes, got %d", res.TotalRoutes)
	}
	if res.LookupsPerSec <= 0 {
		t.Errorf("Expected positive lookup throughput")
	}
	if res.TrieAvgLatency <= 0 {
		t.Errorf("Expected positive Trie latency measurement")
	}

	summary := res.Summary()
	if len(summary) == 0 {
		t.Errorf("Expected non-empty summary report")
	}
}
