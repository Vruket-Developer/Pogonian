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

// AvailableReleases lists all Pogonian releases in descending order (latest on top).
var AvailableReleases = []FrameworkRelease{
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
}

// RunVersion handles `pogn version`, `pogn version -list`, `pogn version --list`.
func RunVersion(args []string) error {
	for _, arg := range args {
		if arg == "-list" || arg == "--list" || arg == "-l" || arg == "list" {
			projectDir := "."
			if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
				projectDir = args[1]
			}
			return RunVersionList(projectDir)
		}
	}

	fmt.Printf("Pogonian CLI v%s\n", Version)
	return nil
}

// RunVersionList displays all Pogonian versions till date (newest on top), highlighting Current, Latest, and (max upgradable).
func RunVersionList(projectDir string) error {
	absDir, _ := filepath.Abs(projectDir)
	currentVersion := Version
	cfgPath := filepath.Join(absDir, "pogonian.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		if cfg, err := config.Load(absDir); err == nil && cfg.Version != "" {
			currentVersion = cfg.Version
		}
	}

	goVer := runtime.Version()
	maxUpgradable := findMaxUpgradableVersion()

	fmt.Println()
	fmt.Printf("📜 \x1b[1;36mPogonian Version History & Compatibility Matrix\x1b[0m\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Current Project Version: \x1b[1;33mv%s\x1b[0m\n", currentVersion)
	fmt.Printf("  Active CLI Binary:       \x1b[1;32mv%s\x1b[0m\n", Version)
	fmt.Printf("  System Go Toolchain:     \x1b[1m%s\x1b[0m\n", goVer)
	fmt.Printf("  Max Upgradable Version:  \x1b[1;35mv%s\x1b[0m \x1b[1;33m(max upgradable)\x1b[0m\n", maxUpgradable)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	for _, rel := range AvailableReleases {
		isCurrent := rel.Version == currentVersion
		isLatest := rel.Channel == ChannelLatest
		isMaxUpgradable := rel.Version == maxUpgradable

		var tags []string
		if isLatest {
			tags = append(tags, "\x1b[1;32m[Latest]\x1b[0m")
		}
		if isCurrent {
			tags = append(tags, "\x1b[1;36m[Current]\x1b[0m")
		}
		if isMaxUpgradable {
			tags = append(tags, "\x1b[1;33m(max upgradable)\x1b[0m")
		}

		tagStr := strings.Join(tags, " ")
		if tagStr != "" {
			tagStr = " " + tagStr
		}

		compatStatus := "\x1b[1;32m✓ Compatible\x1b[0m"
		if !isToolchainCompatible(rel.MinGoVersion) {
			compatStatus = fmt.Sprintf("\x1b[1;31m❌ Incompatible (Requires Go %s+)\x1b[0m", rel.MinGoVersion)
		}

		fmt.Printf("  • \x1b[1;37mv%-10s\x1b[0m [%-10s] — %-12s %s%s\n", rel.Version, rel.Channel, rel.ReleaseDate, compatStatus, tagStr)
		fmt.Printf("    \x1b[1;30m↳ Requires Go: %s+ | Runtime: Pogonian %s\x1b[0m\n", rel.MinGoVersion, rel.Version)

		if len(rel.NewFeatures) > 0 {
			fmt.Printf("    \x1b[1;32m↳ Features:\x1b[0m %s\n", strings.Join(rel.NewFeatures, "; "))
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
	if currentVersion == maxUpgradable {
		fmt.Printf("  ✨ \x1b[1;32mYour project is already at the max upgradable version (v%s)!\x1b[0m\n", currentVersion)
	} else {
		fmt.Printf("  🚀 \x1b[1;33mTo upgrade your project to v%s (max upgradable):\x1b[0m\n", maxUpgradable)
		fmt.Printf("     \x1b[1;32mpogn upgrade --target=%s\x1b[0m\n", maxUpgradable)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func findMaxUpgradableVersion() string {
	for _, rel := range AvailableReleases {
		if isToolchainCompatible(rel.MinGoVersion) {
			return rel.Version
		}
	}
	return Version
}

func isToolchainCompatible(minGo string) bool {
	goVer := runtime.Version()
	if strings.HasPrefix(goVer, "go") {
		goVerNum := strings.TrimPrefix(goVer, "go")
		if minGo > goVerNum && !strings.HasPrefix(goVerNum, minGo) {
			return false
		}
	}
	return true
}

// RunUpgrade executes `pogn upgrade` or `pogn upgrade --check` to verify compatibility before applying changes.
func RunUpgrade(args []string) error {
	checkOnly := false
	dryRun := false
	force := false
	targetVersion := Version
	projectDir := "."

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--check" || arg == "--list" || arg == "-c" || arg == "check":
			checkOnly = true
		case arg == "--dry-run" || arg == "-n":
			dryRun = true
		case arg == "--force" || arg == "-f":
			force = true
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
		return RunVersionList(absDir)
	}

	// Check if already at latest version
	isLatest := currentVersion == targetVersion

	if isLatest && !force && !dryRun {
		fmt.Println()
		fmt.Printf("✨ \x1b[1;32mProject is already at the latest version (v%s)!\x1b[0m\n", currentVersion)
		fmt.Printf("   \x1b[1;30m↳ No upgrade needed.\x1b[0m\n")
		fmt.Printf("   \x1b[1;30m↳ To force re-sync configuration and schemas, run: pogn upgrade --force\x1b[0m\n\n")
		return nil
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
		fmt.Printf("   \x1b[1;30mTo inspect version requirements, run: pogn version -list\x1b[0m\n\n")
		return compatErr
	}

	if dryRun {
		fmt.Println("🔍 \x1b[1;33m[Dry Run Mode] Previewing upgrade actions:\x1b[0m")
		fmt.Printf("  • Would upgrade pogonian.yaml from v%s to v%s\n", currentVersion, targetVersion)
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
			if changed || force {
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
			if changed || force {
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

func verifySystemCompatibility(targetVersion string) error {
	for _, rel := range AvailableReleases {
		if rel.Version == targetVersion {
			if !isToolchainCompatible(rel.MinGoVersion) {
				return fmt.Errorf("target version v%s requires Go %s+, but current toolchain is %s",
					targetVersion, rel.MinGoVersion, runtime.Version())
			}
			return nil
		}
	}
	return nil
}
