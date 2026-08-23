package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"pogonian/internal/migration"
)

// RunImport executes `pogn import <source-dir>`.
func RunImport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("source directory is required: pogn import <path-to-project>")
	}

	sourceDir, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' does not exist", sourceDir)
	}

	_, report, err := migration.MigrateReactProject(sourceDir)
	if err != nil {
		return fmt.Errorf("migration analysis failed: %w", err)
	}

	fmt.Print(report.Summary())
	fmt.Printf("To execute this migration into a fresh Pogonian application, run:\n")
	fmt.Printf("  \x1b[1;36mpogn migrate %s ./migrated-app\x1b[0m\n\n", args[0])

	return nil
}

// RunMigrate executes `pogn migrate <source-dir> [target-dir]`.
func RunMigrate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("source directory is required: pogn migrate <path-to-project> [target-dir]")
	}

	sourceDir, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	targetDir := "migrated-" + filepath.Base(sourceDir)
	if len(args) > 1 {
		targetDir = args[1]
	}

	targetAbs, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	fmt.Printf("🚀 \x1b[1;36mMigrating project from:\x1b[0m %s ➔ %s...\n", sourceDir, targetDir)

	appIR, report, err := migration.MigrateReactProject(sourceDir)
	if err != nil {
		return fmt.Errorf("migration analysis failed: %w", err)
	}

	err = migration.GenerateFromIR(appIR, targetAbs)
	if err != nil {
		return fmt.Errorf("migration generation failed: %w", err)
	}

	fmt.Print(report.Summary())
	fmt.Printf("✨ \x1b[1;32mMigration complete!\x1b[0m Your new Pogonian app is ready at %s\n\n", targetDir)
	fmt.Printf("To start developing:\n")
	fmt.Printf("  cd %s\n", targetDir)
	fmt.Printf("  pogn dev\n\n")

	return nil
}
