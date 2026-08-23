package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pogonian/internal/ai"
)

// RunAdd handles `pogn add table <name>`, `pogn add module <name>`, `pogn add route <path>`.
func RunAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage:\n  pogn add table <name>\n  pogn add module <name>\n  pogn add route <path>")
	}

	targetType := "table"
	targetName := ""
	projectDir := "."

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "table" || arg == "module" || arg == "route" {
			targetType = arg
			if i+1 < len(args) {
				targetName = args[i+1]
				i++
			}
		} else if strings.HasPrefix(arg, "--dir=") {
			projectDir = strings.TrimPrefix(arg, "--dir=")
		} else if targetName == "" {
			targetName = arg
		}
	}

	if targetName == "" {
		return fmt.Errorf("name or path is required: pogn add %s <name>", targetType)
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	if targetType == "route" {
		routePath := strings.TrimPrefix(targetName, "/")
		routeDir := filepath.Join(absDir, "app", routePath)
		if err := os.MkdirAll(routeDir, 0755); err != nil {
			return err
		}

		pageFile := filepath.Join(routeDir, "page.html")
		title := filepath.Base(routeDir)
		titlePascal := strings.ToUpper(title[:1]) + title[1:]

		htmlContent := fmt.Sprintf(`<section class="content-section">
    <h1>%s</h1>
    <p style="color: var(--text-secondary);">Standalone page route created with <code>pogn add route /%s</code></p>
    <div class="card" style="margin-top: 1.5rem;">
        <p>Edit this page at <code>app/%s/page.html</code></p>
    </div>
</section>
`, titlePascal, routePath, routePath)

		if err := os.WriteFile(pageFile, []byte(htmlContent), 0644); err != nil {
			return err
		}

		fmt.Println()
		fmt.Printf("✨ \x1b[1;32mAdded route:\x1b[0m \x1b[1m/%s\x1b[0m\n", routePath)
		fmt.Printf("   • File: \x1b[1;36mapp/%s/page.html\x1b[0m\n", routePath)
		fmt.Printf("   • Live: \x1b[1;34mhttp://localhost:3000/%s\x1b[0m\n\n", routePath)
		return nil
	}

	// For table or module -> scaffold complete full-stack tier
	plan := &ai.AIPlan{
		FeatureName: targetName,
	}

	artifacts, err := ai.ApplyPlan(absDir, plan)
	if err != nil {
		return fmt.Errorf("failed to add %s: %w", targetType, err)
	}

	fmt.Println()
	fmt.Printf("✨ \x1b[1;32mSuccessfully added %s '%s':\x1b[0m\n", targetType, artifacts.ModuleName)
	for _, f := range artifacts.Files {
		rel, _ := filepath.Rel(absDir, f)
		fmt.Printf("  ✓ Created \x1b[1;36m%s\x1b[0m\n", rel)
	}
	fmt.Printf("  ✓ Registered in \x1b[1;36m.pogonian/architecture/app.ai\x1b[0m\n")
	fmt.Printf("  ✓ Dynamic SQLite table ready\n\n")
	fmt.Printf("🚀 \x1b[1mView live at:\x1b[0m \x1b[1;34mhttp://localhost:3000/%s\x1b[0m\n\n", artifacts.ModuleName)

	return nil
}

// RunRemove handles `pogn remove table <name>`, `pogn remove module <name>`, `pogn remove route <path>`.
func RunRemove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage:\n  pogn remove table <name>\n  pogn remove module <name>\n  pogn remove route <path>")
	}

	targetType := "table"
	targetName := ""
	projectDir := "."

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "table" || arg == "module" || arg == "route" {
			targetType = arg
			if i+1 < len(args) {
				targetName = args[i+1]
				i++
			}
		} else if strings.HasPrefix(arg, "--dir=") {
			projectDir = strings.TrimPrefix(arg, "--dir=")
		} else if targetName == "" {
			targetName = arg
		}
	}

	if targetName == "" {
		return fmt.Errorf("name or path is required: pogn remove %s <name>", targetType)
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	mod := strings.ToLower(strings.Trim(targetName, "/"))

	if targetType == "route" {
		routeDir := filepath.Join(absDir, "app", mod)
		if err := os.RemoveAll(routeDir); err != nil {
			return err
		}
		fmt.Println()
		fmt.Printf("🗑️  \x1b[1;31mRemoved route:\x1b[0m \x1b[1m/%s\x1b[0m (deleted app/%s)\n\n", mod, mod)
		return nil
	}

	// Remove server/<mod> and app/<mod>
	serverDir := filepath.Join(absDir, "server", mod)
	appDir := filepath.Join(absDir, "app", mod)

	_ = os.RemoveAll(serverDir)
	_ = os.RemoveAll(appDir)

	// Clean from .pogonian/architecture/app.ai
	aiPath := filepath.Join(absDir, ".pogonian", "architecture", "app.ai")
	if data, err := os.ReadFile(aiPath); err == nil {
		lines := strings.Split(string(data), "\n")
		var newLines []string
		for _, line := range lines {
			if !strings.Contains(line, "- "+mod) && !strings.Contains(line, "- \""+mod+"\"") {
				newLines = append(newLines, line)
			}
		}
		_ = os.WriteFile(aiPath, []byte(strings.Join(newLines, "\n")), 0644)
	}

	fmt.Println()
	fmt.Printf("🗑️  \x1b[1;31mRemoved %s '%s':\x1b[0m\n", targetType, mod)
	fmt.Printf("  ✓ Deleted \x1b[1;36mserver/%s/\x1b[0m\n", mod)
	fmt.Printf("  ✓ Deleted \x1b[1;36mapp/%s/\x1b[0m\n", mod)
	fmt.Printf("  ✓ Unregistered from \x1b[1;36m.pogonian/architecture/app.ai\x1b[0m\n\n")

	return nil
}
