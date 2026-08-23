package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pogonian/internal/config"
	"pogonian/internal/database"
)

// RunClean executes `pogn clean`.
func RunClean(args []string) error {
	projectDir := "."
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	dirsToClean := []string{
		filepath.Join(absDir, "dist"),
		filepath.Join(absDir, ".pogonian", "build"),
		filepath.Join(absDir, ".pogonian", "cache"),
		filepath.Join(absDir, ".test-work"),
	}

	fmt.Println()
	fmt.Printf("🧹 \x1b[1mCleaning Pogonian Project:\x1b[0m %s\n", absDir)
	for _, dir := range dirsToClean {
		if err := os.RemoveAll(dir); err == nil {
			rel, _ := filepath.Rel(absDir, dir)
			fmt.Printf("  ✓ Removed \x1b[1;36m%s\x1b[0m\n", rel)
		}
	}

	fmt.Println("✨ Project workspace is clean!")
	return nil
}

// RunReset executes `pogn reset`.
func RunReset(args []string) error {
	projectDir := "."
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	// 1. Run Clean
	_ = RunClean(args)

	// 2. Reset development database
	cfg, err := config.Load(absDir)
	if err == nil && cfg.Database.Driver == "sqlite" && cfg.Database.Path != "" {
		dbPath := cfg.Database.Path
		if !filepath.IsAbs(dbPath) && dbPath != ":memory:" {
			dbPath = filepath.Join(absDir, dbPath)
		}

		_ = os.Remove(dbPath)
		_ = os.Remove(dbPath + "-wal")
		_ = os.Remove(dbPath + "-shm")

		// Recreate fresh SQLite database
		if db, err := database.Open(dbPath); err == nil {
			_ = db.Close()
			fmt.Printf("  ✓ Reset development database: \x1b[1;32m%s\x1b[0m\n", cfg.Database.Path)
		}
	}

	fmt.Println("✨ \x1b[1;32mProject environment reset to fresh state!\x1b[0m")
	return nil
}
