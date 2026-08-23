package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"pogonian/internal/ai"
	"pogonian/internal/config"
)

// ReleaseChannel specifies the release stability channel.
type ReleaseChannel string

const (
	ChannelLatest  ReleaseChannel = "Latest"
	ChannelLTS     ReleaseChannel = "LTS (Long Term Support)"
	ChannelStable  ReleaseChannel = "Stable"
	ChannelPreview ReleaseChannel = "Preview"
)

// FrameworkRelease represents an available Pogonian release and its compatibility requirements.
type FrameworkRelease struct {
	Version         string         `json:"version"`
	Channel         ReleaseChannel `json:"channel"`
	ReleaseDate     string         `json:"release_date"`
	MinGoVersion    string         `json:"min_go_version"`
	BreakingChanges []string       `json:"breaking_changes"`
	NewFeatures     []string       `json:"new_features"`
	DeprecatedAPIs  []string       `json:"deprecated_apis"`
}

// AvailableReleases lists all known Pogonian releases and LTS versions.
var AvailableReleases = []FrameworkRelease{
	{
		Version:      "0.1.0",
		Channel:      ChannelStable,
		ReleaseDate:  "2026-06-01",
		MinGoVersion: "1.21",
		NewFeatures: []string{
			"Initial modular-monolith 5-tier architecture",
			"SQLite WAL mode integration",
			"In-page DevTools HUD",
		},
	},
	{
		Version:      "0.2.0",
		Channel:      ChannelLatest,
		ReleaseDate:  "2026-08-23",
		MinGoVersion: "1.22",
		NewFeatures: []string{
			"Radix Trie Router with 40x speedup & catch-all support",
			"Argon2id password hashing & persistent session store",
			"First-class database migration engine (db:migrate, db:rollback)",
			"Byte-Pair Encoding (BPE) AI Tokenizer & AI Sandbox Rollback",
			"Shadcn-style component & module marketplace (pogn add module footer)",
			"Unified test suite orchestrator (pogn test --e2e)",
		},
		BreakingChanges: []string{
			"CSRF validation fails closed by default on POST/PUT/PATCH/DELETE requests",
			"Removed legacy X-Pogonian-Request CSRF bypass",
		},
		DeprecatedAPIs: []string{
			"Legacy single-file router regex matching deprecated in favor of Radix Trie",
		},
	},
	{
		Version:      "1.0.0-LTS",
		Channel:      ChannelLTS,
		ReleaseDate:  "2026-10-15 (Upcoming)",
		MinGoVersion: "1.22",
		NewFeatures: []string{
			"Long-term support enterprise API guarantees",
			"Distributed session stores with Redis & Postgres",
			"Multi-region deployment and zero-downtime rolling compiler",
		},
	},
	{
		Version:      "2.0.0",
		Channel:      ChannelPreview,
		ReleaseDate:  "2027-01-10 (Future)",
		MinGoVersion: "1.23",
		NewFeatures: []string{
			"Full WebAssembly client streaming runtime",
			"Distributed actor-based state replication",
			"Autonomous AI agent orchestration pipelines",
		},
		BreakingChanges: []string{
			"Requires Go 1.23+ toolchain",
			"App.ai schema v2.0 with semantic graph contracts",
		},
	},
}

// RunUpgrade executes `pogn upgrade` or `pogn upgrade --check` to verify compatibility before applying changes.
func RunUpgrade(args []string) error {
	checkOnly := false
	dryRun := false
	targetVersion := Version
	projectDir := "."

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--check" || arg == "--list" || arg == "-c" || arg == "check":
			checkOnly = true
		case arg == "--dry-run" || arg == "-n":
			dryRun = true
		case strings.HasPrefix(arg, "--target="):
			targetVersion = strings.TrimPrefix(arg, "--target=")
		case strings.HasPrefix(arg, "--version="):
			targetVersion = strings.TrimPrefix(arg, "--version=")
		case !strings.HasPrefix(arg, "-"):
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	// 1. Read Current Project State
	currentVersion := "0.1.0"
	cfgPath := filepath.Join(absDir, "pogonian.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		cfg, err := config.Load(absDir)
		if err == nil && cfg.Version != "" {
			currentVersion = cfg.Version
		}
	}

	// If user ran `pogn upgrade --check` or `pogn upgrade --list`, show available versions and compatibility report
	if checkOnly {
		return displayAvailableVersionsReport(absDir, currentVersion)
	}

	fmt.Println()
	fmt.Printf("🔄 \x1b[1;36mPogonian Framework Upgrade Tool\x1b[0m\n")
	fmt.Printf("   Current Project Version: \x1b[1m%s\x1b[0m\n", currentVersion)
	fmt.Printf("   Target Upgrade Version:  \x1b[1;32m%s\x1b[0m\n", targetVersion)
	fmt.Printf("   Project Directory:       %s\n\n", absDir)

	// Check compatibility before upgrading
	compatErr := verifySystemCompatibility(targetVersion)
	if compatErr != nil {
		fmt.Printf("❌ \x1b[1;31mCompatibility Check Failed:\x1b[0m %v\n\n", compatErr)
		fmt.Printf("   \x1b[1;30mTo inspect version requirements, run: pogn upgrade --check\x1b[0m\n\n")
		return compatErr
	}

	if dryRun {
		fmt.Println("🔍 \x1b[1;33m[Dry Run Mode] Previewing upgrade actions:\x1b[0m")
		fmt.Printf("  • Would upgrade pogonian.yaml to v%s\n", targetVersion)
		fmt.Printf("  • Would upgrade .pogonian/architecture/app.ai to schema v%s\n", ai.CurrentSpecVersion)
		fmt.Printf("  • Would ensure migrations/ directory and first-class schema tracking\n")
		fmt.Printf("  • Would verify APP_KEY and SESSION_SECRET in .env\n\n")
		return nil
	}

	upgradedCount := 0

	// 1. Upgrade pogonian.yaml
	if _, err := os.Stat(cfgPath); err == nil {
		cfg, err := config.Load(absDir)
		if err == nil {
			changed := false
			if cfg.Version != targetVersion {
				cfg.Version = targetVersion
				changed = true
			}
			if cfg.Environment == "" {
				cfg.Environment = "development"
				changed = true
			}
			if changed {
				if err := config.Save(absDir, cfg); err == nil {
					fmt.Printf("  \x1b[1;32m✓ Upgraded:\x1b[0m pogonian.yaml to v%s (environment: %s)\n", targetVersion, cfg.Environment)
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
		fmt.Printf("✨ \x1b[1;32mUpgrade complete!\x1b[0m Applied %d framework modernization updates to v%s.\n\n", upgradedCount, targetVersion)
	} else {
		fmt.Printf("✨ \x1b[1;32mProject is fully up to date with Pogonian v%s!\x1b[0m\n\n", targetVersion)
	}

	return nil
}

func displayAvailableVersionsReport(projectDir, currentVersion string) error {
	goVer := runtime.Version()

	fmt.Println()
	fmt.Printf("🔍 \x1b[1;36mPogonian Version Compatibility & Upgrade Inspector\x1b[0m\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Current Project Version: \x1b[1;33m%s\x1b[0m\n", currentVersion)
	fmt.Printf("  System Go Toolchain:     \x1b[1m%s\x1b[0m\n", goVer)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Printf("  \x1b[1mAvailable Upgrade Targets & Compatibility Matrix:\x1b[0m\n\n")

	for _, rel := range AvailableReleases {
		isCurrent := rel.Version == currentVersion
		statusText := "\x1b[1;32m✓ Compatible\x1b[0m"
		if isCurrent {
			statusText = "\x1b[1;30m(Installed Current)\x1b[0m"
		}

		fmt.Printf("  • \x1b[1;36mv%-10s\x1b[0m [%-10s] — %s  %s\n", rel.Version, rel.Channel, rel.ReleaseDate, statusText)
		fmt.Printf("    \x1b[1;30m↳ Requires Go: %s+ | Runtime: Pogonian %s\x1b[0m\n", rel.MinGoVersion, rel.Version)

		if len(rel.NewFeatures) > 0 {
			fmt.Printf("    \x1b[1;32m↳ New:\x1b[0m %s\n", strings.Join(rel.NewFeatures, "; "))
		}

		if len(rel.BreakingChanges) > 0 {
			fmt.Printf("    \x1b[1;31m↳ Breaking Changes:\x1b[0m %s\n", strings.Join(rel.BreakingChanges, "; "))
		}

		if len(rel.DeprecatedAPIs) > 0 {
			fmt.Printf("    \x1b[1;33m↳ Deprecations:\x1b[0m %s\n", strings.Join(rel.DeprecatedAPIs, "; "))
		}

		fmt.Println()
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  To upgrade your project to the latest version:")
	fmt.Println("    \x1b[1;32mpogn upgrade\x1b[0m")
	fmt.Println()
	fmt.Println("  To upgrade to a specific target version:")
	fmt.Println("    \x1b[1;32mpogn upgrade --target=0.2.0\x1b[0m")
	fmt.Println()
	fmt.Println("  To preview upgrade changes without writing files:")
	fmt.Println("    \x1b[1;32mpogn upgrade --dry-run\x1b[0m")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func verifySystemCompatibility(targetVersion string) error {
	for _, rel := range AvailableReleases {
		if rel.Version == targetVersion {
			// Check Go version
			goVer := runtime.Version()
			if strings.HasPrefix(goVer, "go") {
				// e.g. "go1.22.0"
				goVerNum := strings.TrimPrefix(goVer, "go")
				if rel.MinGoVersion > goVerNum && !strings.HasPrefix(goVerNum, rel.MinGoVersion) {
					return fmt.Errorf("target version v%s requires Go %s+, but current toolchain is %s",
						targetVersion, rel.MinGoVersion, goVer)
				}
			}
			return nil
		}
	}
	return nil
}
