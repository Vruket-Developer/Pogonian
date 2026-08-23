package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"pogonian/internal/architecture"
)

// RunAnalyze executes `pogn analyze`.
func RunAnalyze(args []string) error {
	projectDir := "."
	format := "ascii"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--format="):
			format = strings.TrimPrefix(arg, "--format=")
		case arg == "-f" || arg == "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case !strings.HasPrefix(arg, "-"):
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	analyzer, err := architecture.NewAnalyzer(absDir)
	if err != nil {
		return err
	}

	appIR, graph, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("architecture analysis failed: %w", err)
	}

	switch strings.ToLower(format) {
	case "json":
		data, err := appIR.ToJSON()
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "mermaid", "mmd":
		fmt.Println(graph.ToMermaid())

	case "yaml", "yml":
		data, err := appIR.ToYAML()
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	default: // ASCII
		fmt.Println()
		fmt.Printf("📊 \x1b[1mArchitecture Graph for:\x1b[0m %s (v%s)\n", appIR.Name, appIR.Version)
		fmt.Printf("   Style: \x1b[1;36m%s\x1b[0m | Database: \x1b[1;33m%s\x1b[0m\n\n", appIR.Architecture, appIR.Database.Driver)
		fmt.Println(graph.ToASCII())
		fmt.Println()
		fmt.Printf("  \x1b[1;30mTip: Run 'pogn analyze --format=mermaid' to export flowchart diagram.\x1b[0m\n\n")
	}

	return nil
}

// RunInspect executes `pogn inspect`.
func RunInspect(args []string) error {
	projectDir := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projectDir = args[0]
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	analyzer, err := architecture.NewAnalyzer(absDir)
	if err != nil {
		return err
	}

	appIR, _, err := analyzer.Analyze()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("🔎 \x1b[1mPogonian Architecture Inspector:\x1b[0m %s\n\n", appIR.Name)

	fmt.Printf("  \x1b[1;34m[1. Frontend Routes (%d)]\x1b[0m\n", len(appIR.Routes))
	for _, r := range appIR.Routes {
		fmt.Printf("    • %-24s (Layouts: %d)\n", r.Path, len(r.LayoutChain))
	}
	fmt.Println()

	fmt.Printf("  \x1b[1;35m[2. Handlers (%d)]\x1b[0m\n", len(appIR.Handlers))
	for _, h := range appIR.Handlers {
		fmt.Printf("    • %-24s (Calls Services: %v)\n", h.Name, h.CallsServices)
	}
	fmt.Println()

	fmt.Printf("  \x1b[1;32m[3. Services (%d)]\x1b[0m\n", len(appIR.Services))
	for _, s := range appIR.Services {
		fmt.Printf("    • %-24s (Calls Repos: %v)\n", s.Name, s.CallsRepos)
	}
	fmt.Println()

	fmt.Printf("  \x1b[1;33m[4. Repositories (%d)]\x1b[0m\n", len(appIR.Repositories))
	for _, repo := range appIR.Repositories {
		fmt.Printf("    • %-24s (Table: %s)\n", repo.Name, repo.TargetTable)
	}
	fmt.Println()

	return nil
}
