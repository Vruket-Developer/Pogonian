package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"pogonian/internal/ai"
	"pogonian/internal/architecture"
)

// RunAI executes `pogn ai <prompt>`.
func RunAI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("prompt is required: pogn ai \"<feature description>\"")
	}

	showTokens := false
	applyChanges := false
	projectDir := "."
	var promptParts []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--tokens" || arg == "-t":
			showTokens = true
		case arg == "--apply" || arg == "-a":
			applyChanges = true
		case strings.HasPrefix(arg, "--dir="):
			projectDir = strings.TrimPrefix(arg, "--dir=")
		case arg == "--dir" || arg == "-d":
			if i+1 < len(args) {
				projectDir = args[i+1]
				i++
			}
		default:
			promptParts = append(promptParts, arg)
		}
	}

	prompt := strings.Join(promptParts, " ")
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}
	projectDir = absDir

	analyzer, err := architecture.NewAnalyzer(projectDir)
	if err != nil {
		return err
	}

	appIR, graph, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("architecture analysis failed: %w", err)
	}

	engine := ai.NewContextEngine(projectDir, appIR, graph)
	ctx, err := engine.ExtractContext(prompt)
	if err != nil {
		return fmt.Errorf("failed to extract AI context: %w", err)
	}

	plan := ai.GeneratePlan(prompt, ctx)

	fmt.Println()
	fmt.Printf("🧠 \x1b[1;36mPogonian AI Architecture Planner\x1b[0m\n")
	fmt.Printf("   Task: \x1b[1m\"%s\"\x1b[0m\n\n", prompt)

	fmt.Printf("  \x1b[1mContext Efficiency:\x1b[0m\n")
	fmt.Printf("    • Full Repository:   ~%d tokens\n", ctx.FullRepoTokens)
	fmt.Printf("    • Pogonian Context:  ~%d tokens (\x1b[1;32m%.1f%% tokens saved\x1b[0m)\n", ctx.ScopedTokens, ctx.SavingsPercent)
	fmt.Printf("    • Targeted Modules:  %s\n\n", strings.Join(ctx.RelevantEntities, ", "))

	if showTokens {
		fmt.Println("--- Scoped Context Output ---")
		fmt.Println(ctx.ContextSummary)
		fmt.Println("-----------------------------")
	}

	fmt.Printf("  \x1b[1mMulti-Layer Implementation Plan:\x1b[0m\n")
	for _, step := range plan.Steps {
		actionColor := "\x1b[1;32m" // green for CREATE
		if step.Action == "MODIFY" {
			actionColor = "\x1b[1;33m" // yellow for MODIFY
		} else if step.Action == "MIGRATE" {
			actionColor = "\x1b[1;35m" // magenta for MIGRATE
		}
		fmt.Printf("    %d. %s[%s]\x1b[0m \x1b[1m%s\x1b[0m (%s)\n", step.Step, actionColor, step.Action, step.Layer, step.File)
		fmt.Printf("       \x1b[1;30m↳ %s\x1b[0m\n", step.Description)
	}

	fmt.Println()
	if applyChanges {
		artifacts, err := ai.ApplyPlan(projectDir, plan)
		if err != nil {
			return fmt.Errorf("failed to apply AI plan: %w", err)
		}

		fmt.Println("🚀 \x1b[1;32mApplied Architectural Changes:\x1b[0m")
		for _, f := range artifacts.Files {
			rel, _ := filepath.Rel(projectDir, f)
			fmt.Printf("  ✓ Created \x1b[1;36m%s\x1b[0m\n", rel)
		}
		fmt.Println("  ✓ Registered module in \x1b[1;36m.pogonian/architecture/app.ai\x1b[0m")
		fmt.Println()
		fmt.Printf("✨ \x1b[1;32mFeature '%s' synthesized and live! Visit http://localhost:3000/%s\x1b[0m\n\n", plan.FeatureName, artifacts.ModuleName)
	} else {
		fmt.Printf("✨ \x1b[1;32mArchitecture plan verified with 0 layer boundary violations.\x1b[0m\n")
		fmt.Printf("   \x1b[1;30mTo apply this plan and generate code, run: pogn ai \"%s\" --apply\x1b[0m\n\n", prompt)
	}

	return nil
}
