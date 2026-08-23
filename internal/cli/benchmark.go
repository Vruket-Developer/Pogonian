package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"pogonian/internal/benchmark"
)

// RunBenchmark executes `pogn benchmark [path]` or `pogn benchmark --stress`.
func RunBenchmark(args []string) error {
	projectDir := "."
	stressMode := false
	numRoutes := 10000
	numLookups := 100000

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--stress" || arg == "-s":
			stressMode = true
		case strings.HasPrefix(arg, "--routes="):
			if n, err := strconv.Atoi(strings.TrimPrefix(arg, "--routes=")); err == nil && n > 0 {
				numRoutes = n
			}
		case strings.HasPrefix(arg, "--lookups="):
			if n, err := strconv.Atoi(strings.TrimPrefix(arg, "--lookups=")); err == nil && n > 0 {
				numLookups = n
			}
		case !strings.HasPrefix(arg, "-"):
			projectDir = arg
		}
	}

	if stressMode {
		fmt.Println()
		fmt.Printf("🚀 \x1b[1;36mRunning Pogonian Scale & Stress Micro-Benchmark...\x1b[0m\n")
		res, err := benchmark.RunStressBenchmark(numRoutes, numLookups)
		if err != nil {
			return fmt.Errorf("stress benchmark failed: %w", err)
		}
		fmt.Print(res.Summary())
		return nil
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	eval, err := benchmark.RunEvaluation(absDir)
	if err != nil {
		return fmt.Errorf("benchmark evaluation failed: %w", err)
	}

	fmt.Print(eval.Summary())
	return nil
}
