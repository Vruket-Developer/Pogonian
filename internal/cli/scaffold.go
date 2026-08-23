package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pogonian/internal/ai"
	"pogonian/internal/architecture"
	"pogonian/internal/modules"
)

// RunAdd handles `pogn add module <name>`, `pogn add component <name>`, `pogn add table <name>`, `pogn add route <path>`.
func RunAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage:\n  pogn add module <name>\n  pogn add component <name>\n  pogn add table <name>\n  pogn add route <path>")
	}

	targetType := ""
	targetName := ""
	projectDir := "."

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "module", "component", "table", "route":
			targetType = arg
			if i+1 < len(args) {
				targetName = args[i+1]
				i++
			}
		default:
			if strings.HasPrefix(arg, "--dir=") {
				projectDir = strings.TrimPrefix(arg, "--dir=")
			} else if targetName == "" && targetType != "" {
				targetName = arg
			} else if targetType == "" {
				// Check if arg is a known module/component name directly (e.g. `pogn add footer`)
				if _, exists := modules.Catalog[strings.ToLower(arg)]; exists {
					targetType = "module"
					targetName = arg
				} else {
					targetType = "table"
					targetName = arg
				}
			}
		}
	}

	if targetName == "" {
		return fmt.Errorf("name or path is required: pogn add %s <name>", targetType)
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	// 1. Module / Component Marketplace Scaffolding
	if targetType == "module" || targetType == "component" {
		res, err := modules.InstallModule(absDir, targetName)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Printf("📦 \x1b[1;36mPogonian Marketplace Installer\x1b[0m\n")
		fmt.Printf("   Successfully installed \x1b[1;32m%s\x1b[0m (\x1b[1m%s\x1b[0m)\n\n", res.ModuleName, res.Category)

		for _, f := range res.FilesCreated {
			rel, _ := filepath.Rel(absDir, f)
			fmt.Printf("  ✓ Created \x1b[1;36m%s\x1b[0m\n", rel)
		}

		if res.UsageHint != "" {
			fmt.Println()
			fmt.Printf("💡 \x1b[1mHow to use in your pages:\x1b[0m\n")
			fmt.Printf("   \x1b[1;33m%s\x1b[0m\n", res.UsageHint)
		}
		fmt.Println()
		return nil
	}

	// 2. Standalone Route Page Scaffolding
	if targetType == "route" {
		routePath := strings.TrimPrefix(targetName, "/")
		routeDir := filepath.Join(absDir, "app", routePath)
		if err := os.MkdirAll(routeDir, 0755); err != nil {
			return err
		}

		pageFile := filepath.Join(routeDir, "page.html")
		title := filepath.Base(routeDir)
		titlePascal := strings.ToUpper(title[:1]) + title[1:]

		// Check if matching Go entity exists in server/<routePath>/entity.go
		entityPath := filepath.Join(absDir, "server", routePath, "entity.go")
		var htmlContent string

		if _, err := os.Stat(entityPath); err == nil {
			structName, fields, err := architecture.ParseEntityFile(entityPath)
			if err == nil && len(fields) > 0 {
				htmlContent = architecture.GenerateHTMLFromFields(routePath, structName, fields)
				fmt.Println()
				fmt.Printf("🧬 \x1b[1;35mSchema-Driven AST Generator:\x1b[0m Inspected \x1b[1mserver/%s/entity.go\x1b[0m\n", routePath)
				fmt.Printf("   • Mapped %d struct fields to HTML inputs & table columns\n", len(fields))
			}
		}

		if htmlContent == "" {
			htmlContent = fmt.Sprintf(`<section class="content-section">
    <h1>%s</h1>
    <p style="color: var(--text-secondary);">Standalone page route created with <code>pogn add route /%s</code></p>
    <div class="card" style="margin-top: 1.5rem;">
        <p>Edit this page at <code>app/%s/page.html</code></p>
    </div>
</section>
`, titlePascal, routePath, routePath)
		}

		if err := os.WriteFile(pageFile, []byte(htmlContent), 0644); err != nil {
			return err
		}

		fmt.Println()
		fmt.Printf("✨ \x1b[1;32mGenerated route:\x1b[0m \x1b[1m/%s\x1b[0m\n", routePath)
		fmt.Printf("   • File: \x1b[1;36mapp/%s/page.html\x1b[0m\n", routePath)
		fmt.Printf("   • Live: \x1b[1;34mhttp://localhost:3000/%s\x1b[0m\n\n", routePath)
		return nil
	}

	// 3. Database Table Entity Scaffolding (Full-stack Go CRUD + View)
	plan := &ai.AIPlan{
		FeatureName: targetName,
	}

	artifacts, err := ai.ApplyPlan(absDir, plan)
	if err != nil {
		return fmt.Errorf("failed to add table: %w", err)
	}

	fmt.Println()
	fmt.Printf("✨ \x1b[1;32mSuccessfully added table '%s':\x1b[0m\n", artifacts.ModuleName)
	for _, f := range artifacts.Files {
		rel, _ := filepath.Rel(absDir, f)
		fmt.Printf("  ✓ Created \x1b[1;36m%s\x1b[0m\n", rel)
	}
	fmt.Printf("  ✓ Registered in \x1b[1;36m.pogonian/architecture/app.ai\x1b[0m\n")
	fmt.Printf("  ✓ Dynamic SQLite table ready\n\n")
	fmt.Printf("🚀 \x1b[1mView live at:\x1b[0m \x1b[1;34mhttp://localhost:3000/%s\x1b[0m\n\n", artifacts.ModuleName)

	return nil
}

// RunListModules displays the catalog of all available marketplace components.
func RunListModules(args []string) error {
	fmt.Println()
	fmt.Printf("🏪 \x1b[1;36mPogonian Component & Module Marketplace\x1b[0m\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Add any component/module directly to your project without npm dependencies:")
	fmt.Println("    \x1b[1;33mpogn add module <name>\x1b[0m  or  \x1b[1;33mpogn add component <name>\x1b[0m")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	mods := modules.ListModules()
	for _, m := range mods {
		typeColor := "\x1b[1;34m" // UI
		if m.Category == modules.CategoryFullStack {
			typeColor = "\x1b[1;35m" // Full stack
		}

		fmt.Printf("  • %-12s %s[%s]\x1b[0m\n", "\x1b[1;36m"+m.Name+"\x1b[0m", typeColor, m.Category)
		fmt.Printf("    \x1b[1;30m↳ %s\x1b[0m\n", m.Description)
		if m.UsageHint != "" {
			fmt.Printf("    \x1b[1;32m↳ %s\x1b[0m\n", m.UsageHint)
		}
		fmt.Println()
	}

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
		if arg == "table" || arg == "module" || arg == "route" || arg == "component" {
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

	// Component removal
	if targetType == "component" {
		compFile := filepath.Join(absDir, "app", "components", mod+".html")
		_ = os.Remove(compFile)
		fmt.Println()
		fmt.Printf("🗑️  \x1b[1;31mRemoved component:\x1b[0m \x1b[1mapp/components/%s.html\x1b[0m\n\n", mod)
		return nil
	}

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
