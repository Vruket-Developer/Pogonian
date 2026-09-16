package cli

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"pogonian/internal/architecture"
	"pogonian/internal/security"
	"pogonian/pkg/config"
	"pogonian/pkg/router"
)

// RunCheck executes `pogn check` / `pogn lint`.
func RunCheck(args []string) error {
	projectDir := "."
	if len(args) > 0 && args[0] != "" {
		projectDir = args[0]
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("🔍 \x1b[1mChecking Pogonian Project:\x1b[0m %s\n\n", absDir)

	passed := true

	// 1. Config Check
	cfg, err := config.Load(absDir)
	if err != nil {
		fmt.Printf("  \x1b[1;31m❌ Configuration:\x1b[0m %v\n", err)
		passed = false
	} else {
		fmt.Printf("  \x1b[1;32m✓ Configuration:\x1b[0m %s (v%s, %s)\n", cfg.Name, cfg.Version, cfg.Architecture.Style)
	}

	// 2. AI Architecture Spec Check
	aiPath := filepath.Join(absDir, ".pogonian", "architecture", "app.ai")
	if _, err := os.Stat(aiPath); err != nil {
		fmt.Printf("  \x1b[1;33m⚠ Architecture Spec:\x1b[0m .pogonian/architecture/app.ai missing\n")
	} else {
		fmt.Printf("  \x1b[1;32m✓ Architecture Spec:\x1b[0m .pogonian/architecture/app.ai\n")
	}

	// 3. Filesystem Routes & Templates Check
	appDir := filepath.Join(absDir, "app")
	r, err := router.New(appDir)
	if err != nil {
		fmt.Printf("  \x1b[1;31m❌ Filesystem Router:\x1b[0m %v\n", err)
		passed = false
	} else {
		fmt.Printf("  \x1b[1;32m✓ Filesystem Router:\x1b[0m %d routes discovered\n", len(r.Routes))
		for _, rt := range r.Routes {
			// Check template syntax
			data, err := os.ReadFile(rt.PagePath)
			if err != nil {
				fmt.Printf("    \x1b[1;31m❌ %s (failed to read: %v)\x1b[0m\n", rt.Path, err)
				passed = false
				continue
			}
			_, err = template.New("check").Parse(string(data))
			if err != nil {
				fmt.Printf("    \x1b[1;31m❌ %s (template error: %v)\x1b[0m\n", rt.Path, err)
				passed = false
			} else {
				fmt.Printf("    \x1b[1;30m•\x1b[0m %-20s \x1b[1;32mvalid\x1b[0m\n", rt.Path)
			}
		}
	}

	// 4. Layouts Check
	rootLayout := filepath.Join(appDir, "layout.html")
	if _, err := os.Stat(rootLayout); err != nil {
		fmt.Printf("  \x1b[1;31m❌ Root Layout:\x1b[0m app/layout.html missing\n")
		passed = false
	} else {
		fmt.Printf("  \x1b[1;32m✓ Root Layout:\x1b[0m app/layout.html valid\n")
	}

	// 5. Backend Architecture & Layer Rules Check
	analyzer, err := architecture.NewAnalyzer(absDir)
	if err == nil {
		appIR, graph, err := analyzer.Analyze()
		if err == nil {
			validator := architecture.NewValidator(false)
			violations := validator.Validate(appIR, graph)
			if len(violations) > 0 {
				fmt.Println()
				fmt.Printf("  \x1b[1;33m⚠ Architecture Diagnostics (%d):\x1b[0m\n", len(violations))
				for _, v := range violations {
					if v.Severity == "ERROR" {
						fmt.Printf("    \x1b[1;31m❌ [%s] %s:\x1b[0m %s\n", v.Layer, v.Component, v.Message)
						fmt.Printf("       \x1b[1;30m↳ Suggestion: %s\x1b[0m\n", v.Suggestion)
						passed = false
					} else {
						fmt.Printf("    \x1b[1;33m⚠ [%s] %s:\x1b[0m %s\n", v.Layer, v.Component, v.Message)
					}
				}
			} else {
				fmt.Printf("  \x1b[1;32m✓ Architecture Layers:\x1b[0m 0 violations detected (Clean Flow)\n")
			}
		}
	}

	// 6. Security Policy Rules Check
	report, err := security.AuditProject(absDir)
	if err == nil && report != nil {
		highSeverityCount := 0
		for _, f := range report.Findings {
			if f.Severity == "HIGH" || f.Severity == "CRITICAL" {
				highSeverityCount++
			}
		}
		if highSeverityCount > 0 {
			fmt.Println()
			fmt.Printf("  \x1b[1;31m❌ Security Policy Violations (%d):\x1b[0m\n", highSeverityCount)
			for _, f := range report.Findings {
				if f.Severity == "HIGH" || f.Severity == "CRITICAL" {
					fmt.Printf("    \x1b[1;31m❌ [%s] %s (%s:%d):\x1b[0m %s\n", f.Severity, f.Type, f.File, f.Line, f.Description)
				}
			}
			passed = false
		} else {
			fmt.Printf("  \x1b[1;32m✓ Security Policy:\x1b[0m 0 high-severity vulnerabilities detected\n")
		}
	}

	fmt.Println()
	if passed {
		fmt.Println("✨ \x1b[1;32mAll architecture and route checks passed!\x1b[0m")
		return nil
	} else {
		return fmt.Errorf("architecture check failed with errors")
	}
}


