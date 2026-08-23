package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"pogonian/internal/ai"
	"pogonian/internal/config"
)

// RunUpgrade executes `pogn upgrade` to migrate project configs, .ai schemas, and runtime to the latest Pogonian version.
func RunUpgrade(args []string) error {
	projectDir := "."
	if len(args) > 0 && args[0] != "" {
		projectDir = args[0]
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("🔄 \x1b[1;36mPogonian Framework Upgrade Tool\x1b[0m (Target: v%s)\n", Version)
	fmt.Printf("   Project: %s\n\n", absDir)

	upgradedCount := 0

	// 1. Upgrade pogonian.yaml
	cfgPath := filepath.Join(absDir, "pogonian.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		cfg, err := config.Load(absDir)
		if err == nil {
			changed := false
			if cfg.Version != Version {
				cfg.Version = Version
				changed = true
			}
			if cfg.Environment == "" {
				cfg.Environment = "development"
				changed = true
			}
			if changed {
				if err := config.Save(absDir, cfg); err == nil {
					fmt.Printf("  \x1b[1;32m✓ Upgraded:\x1b[0m pogonian.yaml to v%s (environment: %s)\n", Version, cfg.Environment)
					upgradedCount++
				}
			} else {
				fmt.Printf("  \x1b[1;30m• pogonian.yaml is already up to date (v%s)\x1b[0m\n", cfg.Version)
			}
		}
	}

	// 2. Upgrade .pogonian/architecture/app.ai schema
	aiPath := filepath.Join(absDir, ".pogonian", "architecture", "app.ai")
	if _, err := os.Stat(aiPath); err == nil {
		spec, err := ai.ParseAISpec(absDir)
		if err == nil {
			changed := false
			if spec.Schema != ai.CurrentSpecVersion {
				spec.Schema = ai.CurrentSpecVersion
				changed = true
			}
			if spec.Permissions.RestrictedPaths == nil {
				spec.Permissions = ai.PermissionsSpec{
					AllowCreateFile:      true,
					AllowModifyFile:      true,
					AllowDeleteFile:      false,
					AllowDatabaseMigrate: true,
					RestrictedPaths:      []string{".env", "go.mod", "go.sum", ".git"},
				}
				changed = true
			}
			if changed {
				if err := spec.Save(absDir); err == nil {
					fmt.Printf("  \x1b[1;32m✓ Upgraded:\x1b[0m .pogonian/architecture/app.ai to schema v%s\n", ai.CurrentSpecVersion)
					upgradedCount++
				}
			} else {
				fmt.Printf("  \x1b[1;30m• app.ai specification schema is up to date (v%s)\x1b[0m\n", spec.Schema)
			}
		}
	}

	// 3. Ensure migrations directory exists
	migrationsDir := filepath.Join(absDir, "migrations")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		_ = os.MkdirAll(migrationsDir, 0755)
		fmt.Printf("  \x1b[1;32m✓ Created:\x1b[0m migrations/ directory for first-class schema migrations\n")
		upgradedCount++
	}

	// 4. Ensure .env has APP_KEY and SESSION_SECRET
	if envPath, err := config.EnsureEnvFile(absDir, filepath.Base(absDir)); err == nil && envPath != "" {
		fmt.Printf("  \x1b[1;32m✓ Verified:\x1b[0m Cryptographic secret keys in .env\n")
	}

	fmt.Println()
	if upgradedCount > 0 {
		fmt.Printf("✨ \x1b[1;32mUpgrade complete!\x1b[0m Applied %d framework modernization updates.\n\n", upgradedCount)
	} else {
		fmt.Printf("✨ \x1b[1;32mProject is fully up to date with Pogonian v%s!\x1b[0m\n\n", Version)
	}

	return nil
}
