package cli

import (
	"fmt"
	"path/filepath"

	"pogonian/internal/benchmark"
)

// RunBenchmark executes `pogn benchmark [path]`.
func RunBenchmark(args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	eval, err := benchmark.RunEvaluation(absDir)
	if err != nil {
		return fmt.Errorf("benchmark execution failed: %w", err)
	}

	fmt.Print(eval.Summary())
	return nil
}
