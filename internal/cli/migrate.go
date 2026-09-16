package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pogonian/internal/architecture"
	"pogonian/internal/migration"
	"pogonian/pkg/ir"
	"pogonian/pkg/state"
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

// RunMigrate executes `pogn migrate export`, `pogn migrate import`, or legacy `pogn migrate <source-dir>`.
func RunMigrate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage:\n  pogn migrate export [--route=/path] [--out=file.json]\n  pogn migrate import <payload-file-or-json>\n  pogn migrate <source-dir> [target-dir]")
	}

	sub := args[0]
	switch sub {
	case "export":
		return runMigrateExport(args[1:])
	case "import":
		return runMigrateImport(args[1:])
	default:
		// Legacy project migration
		return runLegacyMigrate(args)
	}
}

func runMigrateExport(args []string) error {
	route := "/"
	outFile := ""
	key := ""
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--route=") {
			route = strings.TrimPrefix(args[i], "--route=")
		} else if strings.HasPrefix(args[i], "--out=") {
			outFile = strings.TrimPrefix(args[i], "--out=")
		} else if (args[i] == "-o" || args[i] == "--out") && i+1 < len(args) {
			outFile = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--key=") {
			key = strings.TrimPrefix(args[i], "--key=")
		} else if (args[i] == "-k" || args[i] == "--key") && i+1 < len(args) {
			key = args[i+1]
			i++
		}
	}

	if key == "" {
		key = os.Getenv("APP_KEY")
		if key == "" {
			key = os.Getenv("SESSION_SECRET")
		}
		if key == "" {
			key = "pogonian-default-dev-transfer-key-32b!"
		}
	}

	projectDir, _ := filepath.Abs(".")
	analyzer, err := architecture.NewAnalyzer(projectDir)
	if err != nil {
		return fmt.Errorf("failed to analyze project architecture: %w", err)
	}
	appIR, _, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("failed to extract project IR: %w", err)
	}

	engine := state.NewEngine(key)
	stateData := map[string]any{
		"client_migrated_at": time.Now().Format(time.RFC3339),
	}

	snapshot, err := engine.CreateMigrationSnapshot(appIR, ir.TargetLinux, route, "", stateData, nil)
	if err != nil {
		return fmt.Errorf("failed to create migration snapshot: %w", err)
	}

	payload, err := engine.PackEnvelope(snapshot, 15*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to pack tamper-proof envelope: %w", err)
	}

	if outFile != "" {
		if err := os.WriteFile(outFile, payload, 0644); err != nil {
			return fmt.Errorf("failed to write envelope to %s: %w", outFile, err)
		}
		fmt.Printf("📦 \x1b[1;32mExported tamper-proof state envelope to\x1b[0m %s (Route: %s)\n", outFile, route)
	} else {
		fmt.Println(string(payload))
	}
	return nil
}

func runMigrateImport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("envelope payload is required: pogn migrate import <payload-file-or-json> [--target=linux|android|ios|android_tv] [--key=...]")
	}

	raw := ""
	key := ""
	target := ir.TargetLinux

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--key=") {
			key = strings.TrimPrefix(args[i], "--key=")
		} else if (args[i] == "-k" || args[i] == "--key") && i+1 < len(args) {
			key = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--target=") {
			target = ir.TargetPlatform(strings.TrimPrefix(args[i], "--target="))
		} else if (args[i] == "-t" || args[i] == "--target") && i+1 < len(args) {
			target = ir.TargetPlatform(args[i+1])
			i++
		} else if raw == "" && !strings.HasPrefix(args[i], "-") {
			raw = args[i]
		}
	}

	if raw == "" {
		return fmt.Errorf("envelope payload is required: pogn migrate import <payload-file-or-json>")
	}

	if key == "" {
		key = os.Getenv("APP_KEY")
		if key == "" {
			key = os.Getenv("SESSION_SECRET")
		}
		if key == "" {
			key = "pogonian-default-dev-transfer-key-32b!"
		}
	}

	var payloadBytes []byte
	if _, err := os.Stat(raw); err == nil {
		data, err := os.ReadFile(raw)
		if err != nil {
			return fmt.Errorf("failed to read envelope file: %w", err)
		}
		payloadBytes = data
	} else {
		payloadBytes = []byte(raw)
	}

	projectDir, _ := filepath.Abs(".")
	analyzer, err := architecture.NewAnalyzer(projectDir)
	if err != nil {
		return fmt.Errorf("failed to analyze project architecture: %w", err)
	}
	appIR, _, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("failed to extract project IR: %w", err)
	}

	engine := state.NewEngine(key)
	snapshot, err := engine.UnpackEnvelope(payloadBytes)
	if err != nil {
		return fmt.Errorf("cryptographic envelope verification failed: %w", err)
	}

	hydrated, err := engine.ReconstructState(snapshot, appIR, target)
	if err != nil {
		return fmt.Errorf("migration state reconstruction failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("✨ \x1b[1;32mApplication Successfully Migrated!\x1b[0m\n")
	fmt.Printf("   Snapshot ID:      %s\n", hydrated.SnapshotID)
	fmt.Printf("   Active Route:     %s\n", hydrated.ActiveRoute)
	fmt.Printf("   Origin Platform:  %s\n", hydrated.OriginPlatform)
	fmt.Printf("   Target Platform:  %s\n", hydrated.TargetPlatform)
	if hydrated.MountedResource != nil {
		fmt.Printf("   Mounted Resource: %s (Entity: %s, Table: %s)\n",
			hydrated.MountedResource.Resource, hydrated.MountedResource.Entity, hydrated.MountedResource.Table)
	}
	if hydrated.AdaptationPlan != nil {
		fmt.Printf("   Input Mode:       %s\n", hydrated.AdaptationPlan.InputMode)
	}
	fmt.Printf("   Restored Fields:  %d fields\n", len(hydrated.RestoredFields))
	if len(hydrated.OperationResults) > 0 {
		fmt.Printf("   Resumed Tasks:    %d active operations continued\n", len(hydrated.OperationResults))
		for _, op := range hydrated.OperationResults {
			fmt.Printf("     • [%s] %s: %s\n", op.Status, op.Type, op.Message)
		}
	}
	fmt.Println()
	return nil
}

func runLegacyMigrate(args []string) error {
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
