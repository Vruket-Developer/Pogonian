package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"pogonian/internal/config"
	"pogonian/internal/database"
)

// RunClean executes `pogn clean` to remove build artifacts, caches, and test runs dynamically.
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

	// Dynamically scan for directories and temporary files to clean
	var itemsRemoved []string

	knownCleanDirs := []string{
		"dist",
		"bin",
		filepath.Join(".pogonian", "build"),
		filepath.Join(".pogonian", "cache"),
		".test-work",
		".pogonian_sandbox",
		".tmp",
	}

	for _, rel := range knownCleanDirs {
		target := filepath.Join(absDir, rel)
		if info, err := os.Stat(target); err == nil {
			if err := os.RemoveAll(target); err == nil {
				if info.IsDir() {
					itemsRemoved = append(itemsRemoved, rel+"/")
				} else {
					itemsRemoved = append(itemsRemoved, rel)
				}
			}
		}
	}

	// Dynamically find and remove any compiled binaries in root (.exe, temporary executables, coverage.out)
	_ = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "templates" {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if name == "coverage.out" || name == "profile.pprof" || strings.HasSuffix(name, ".test") {
			_ = os.Remove(path)
			rel, _ := filepath.Rel(absDir, path)
			itemsRemoved = append(itemsRemoved, rel)
		}
		return nil
	})

	fmt.Println()
	fmt.Printf("🧹 \x1b[1;36mCleaning Pogonian Project:\x1b[0m %s\n", absDir)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if len(itemsRemoved) == 0 {
		fmt.Println("  ✨ Workspace is already clean! No build artifacts or temporary files found.")
	} else {
		for _, item := range itemsRemoved {
			fmt.Printf("  \x1b[1;32m✓ Removed:\x1b[0m \x1b[1;36m%s\x1b[0m\n", item)
		}
		fmt.Println()
		fmt.Printf("✨ Cleaned %d build artifacts & caches.\n", len(itemsRemoved))
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

// RunReset executes `pogn reset` to clean artifacts and reset the database to a fresh state.
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

	// 2. Reset development database dynamically
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
			fmt.Printf("  \x1b[1;32m✓ Re-initialized database:\x1b[0m \x1b[1;36m%s\x1b[0m\n", cfg.Database.Path)
		}
	}

	fmt.Println("✨ \x1b[1;32mProject environment reset to fresh state!\x1b[0m")
	fmt.Println()
	return nil
}
