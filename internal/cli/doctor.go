package cli

import (
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strings"

	"pogonian/pkg/config"
	"pogonian/pkg/dataruntime"
	"pogonian/pkg/router"
	"pogonian/internal/security"
)

// RunDoctor executes system and project diagnostics.
func RunDoctor(args []string) error {
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

	fmt.Println()
	fmt.Println("🩺 \x1b[1mPogonian Environment & Project Doctor\x1b[0m")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. Check Go Toolchain
	goVerCmd := exec.Command("go", "version")
	if out, err := goVerCmd.CombinedOutput(); err == nil {
		fmt.Printf("  ✓ \x1b[1;32mGo Toolchain:\x1b[0m          %s", string(out))
	} else {
		fmt.Printf("  ✗ \x1b[1;31mGo Toolchain:\x1b[0m          Not found in PATH\n")
	}

	// 2. Check Node / npm (optional for tooling)
	nodeVerCmd := exec.Command("node", "--version")
	if out, err := nodeVerCmd.CombinedOutput(); err == nil {
		fmt.Printf("  ✓ \x1b[1;32mNode.js Runtime:\x1b[0m       %s", string(out))
	} else {
		fmt.Printf("  ⚠ \x1b[1;33mNode.js Runtime:\x1b[0m       Not found (optional)\n")
	}

	// 3. Check Pogonian CLI
	fmt.Printf("  ✓ \x1b[1;32mPogonian CLI:\x1b[0m          v%s (Architecture-First)\n", Version)

	// 4. Check Project Configuration
	cfg, err := config.Load(absDir)
	if err != nil {
		fmt.Printf("  ✗ \x1b[1;31mProject Configuration:\x1b[0m Missing or invalid pogonian.yaml\n")
	} else {
		fmt.Printf("  ✓ \x1b[1;32mProject Config:\x1b[0m        %s (v%s, %s)\n", cfg.Name, cfg.Version, cfg.Architecture.Style)

		// 5. Check Database
		if cfg.Database.Driver == "sqlite" {
			dbPath := cfg.Database.Path
			if !filepath.IsAbs(dbPath) && dbPath != ":memory:" {
				dbPath = filepath.Join(absDir, dbPath)
			}
			if db, err := dataruntime.Open(dbPath); err == nil {
				defer db.Close()
				fmt.Printf("  ✓ \x1b[1;32mDatabase (SQLite):\x1b[0m     WAL mode active at %s\n", cfg.Database.Path)
			} else {
				fmt.Printf("  ⚠ \x1b[1;33mDatabase (SQLite):\x1b[0m     Unreachable: %v\n", err)
			}
		} else {
			fmt.Printf("  ✓ \x1b[1;32mDatabase (%s):\x1b[0m     %s\n", cfg.Database.Driver, cfg.Database.URL)
		}

		// 6. Check Port Availability
		portAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		ln, err := net.Listen("tcp", portAddr)
		if err == nil {
			_ = ln.Close()
			fmt.Printf("  ✓ \x1b[1;32mServer Port %d:\x1b[0m        Available for binding\n", cfg.Server.Port)
		} else {
			fmt.Printf("  ℹ \x1b[1;34mServer Port %d:\x1b[0m        In use (active dev server)\n", cfg.Server.Port)
		}
	}

	// 7. Check Router & Filesystem Routes
	appDir := filepath.Join(absDir, "app")
	if r, err := router.New(appDir); err == nil {
		fmt.Printf("  ✓ \x1b[1;32mFilesystem Router:\x1b[0m     %d routes discovered cleanly\n", len(r.Routes))
	} else {
		fmt.Printf("  ⚠ \x1b[1;33mFilesystem Router:\x1b[0m     %v\n", err)
	}

	// 8. Run Quick Security Posture Check
	audit, err := security.AuditProject(absDir)
	if err == nil {
		fmt.Printf("  ✓ \x1b[1;32mSecurity Posture:\x1b[0m      Score %d/100 (Grade %s, %d findings)\n",
			audit.Score, audit.Grade, len(audit.Findings))
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✨ \x1b[1mSystem health check complete!\x1b[0m")
	return nil
}
